#!/usr/bin/env bash
# 在构建机上构建 PandaWiki 全部业务镜像（api / consumer / app / nginx），架构跟随构建机（106 为 arm64）。
# api / consumer 镜像同时包含普通版与雳鉴插桩版（需 backend/iast/go-active-linux-arm64，见 docs/update-operation.md）。
#
# 用法：
#   scripts/build-images.sh                        # 构建全部，tag 为 prod
#   scripts/build-images.sh --only api,consumer    # 只构建部分
#   scripts/build-images.sh --deploy /opt/panda-wiki
#   scripts/build-images.sh --save /opt/packages/panda-wiki-images.tar.gz
#
# 可用环境变量：TAG VERSION GOPROXY NPM_REGISTRY APK_MIRROR
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TAG="${TAG:-prod}"
VERSION="${VERSION:-$(git -C "$ROOT" describe --always --dirty 2>/dev/null || date +%Y%m%d%H%M)}"
GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
NPM_REGISTRY="${NPM_REGISTRY:-https://registry.npmmirror.com}"
APK_MIRROR="${APK_MIRROR:-mirrors.aliyun.com}"
ONLY="api,consumer,app,nginx"
DEPLOY=""
SAVE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --only) ONLY="$2"; shift 2 ;;
    --deploy) DEPLOY="$2"; shift 2 ;;
    --save) SAVE="$2"; shift 2 ;;
    -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
    *) echo "未知参数: $1" >&2; exit 1 ;;
  esac
done

want() { [[ ",${ONLY}," == *",$1,"* ]]; }
log() { echo -e "\n==> $*"; }

for base in golang:1.24.3-alpine openeuler/openeuler:24.03-lts node:22-alpine nginx:alpine; do
  docker image inspect "$base" >/dev/null 2>&1 || { echo "缺少基础镜像 $base（可用 docker.m.daocloud.io 镜像站拉取后 tag）" >&2; exit 1; }
done

STAMP="$(date +%Y%m%d%H%M)"
IMAGES=()
for svc in api consumer app nginx; do
  want "$svc" || continue
  IMAGES+=("panda-wiki-${svc}:${TAG}")
  if docker image inspect "panda-wiki-${svc}:${TAG}" >/dev/null 2>&1; then
    docker tag "panda-wiki-${svc}:${TAG}" "panda-wiki-${svc}:${TAG}-bak-${STAMP}"
  fi
done
log "VERSION=${VERSION} TAG=${TAG} ONLY=${ONLY}（旧镜像已备份为 *:${TAG}-bak-${STAMP}）"

if want api || want consumer; then
  [[ -f "$ROOT/backend/iast/go-active-linux-arm64" ]] \
    || echo "WARN: 缺少 backend/iast/go-active-linux-arm64，镜像将不含插桩版" >&2
fi
for svc in api consumer; do
  want "$svc" || continue
  log "build panda-wiki-${svc}:${TAG}"
  docker build \
    --build-arg "GOPROXY=${GOPROXY}" \
    --build-arg "VERSION=${VERSION}" \
    -f "$ROOT/backend/Dockerfile.${svc}" \
    -t "panda-wiki-${svc}:${TAG}" \
    "$ROOT/backend"
done

if want app || want nginx; then
  log "compile web (node:22-alpine)"
  WEB_TARGETS=""
  want app && WEB_TARGETS="${WEB_TARGETS} panda-wiki-app"
  want nginx && WEB_TARGETS="${WEB_TARGETS} panda-wiki-admin"
  docker run --rm \
    -v "$ROOT/web:/src" \
    -v panda-wiki-npm-cache:/root/.npm \
    -w /src \
    -e "VITE_APP_VERSION=${VERSION}" \
    -e "NEXT_TELEMETRY_DISABLED=1" \
    -e "NPM_CONFIG_REGISTRY=${NPM_REGISTRY}" \
    -e "WEB_TARGETS=${WEB_TARGETS}" \
    node:22-alpine sh -ec "
      sed -i 's#dl-cdn.alpinelinux.org#${APK_MIRROR}#g' /etc/apk/repositories
      apk add --no-cache patch git >/dev/null
      npm ci --no-audit --no-fund
      for w in \${WEB_TARGETS}; do npm run build --workspace=\$w; done
    "
fi
if want app; then
  log "build panda-wiki-app:${TAG}"
  docker build -t "panda-wiki-app:${TAG}" "$ROOT/web/app"
fi
if want nginx; then
  log "build panda-wiki-nginx:${TAG}"
  docker build -t "panda-wiki-nginx:${TAG}" "$ROOT/web/admin"
fi

if [[ -n "$SAVE" ]]; then
  log "save ${IMAGES[*]} -> ${SAVE}"
  mkdir -p "$(dirname "$SAVE")"
  docker save "${IMAGES[@]}" | gzip -1 > "$SAVE"
  ls -lh "$SAVE"
fi

if [[ -n "$DEPLOY" ]]; then
  SERVICES=()
  for svc in api consumer app nginx; do want "$svc" && SERVICES+=("$svc"); done
  log "deploy ${SERVICES[*]} in ${DEPLOY}"
  (cd "$DEPLOY" && docker compose up -d --no-build --pull never --force-recreate "${SERVICES[@]}")
fi

log "完成：${IMAGES[*]}"
