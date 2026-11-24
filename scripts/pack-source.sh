#!/usr/bin/env bash
# 打包交付用源码包（7z / zip）。
#
# 用法：
#   scripts/pack-source.sh                         # 默认 7z → /opt/packages 或 ./packages
#   scripts/pack-source.sh --format zip
#   scripts/pack-source.sh --format zip --out /path/to/dir
#   OUT=/data/packages scripts/pack-source.sh --format zip
#
# ========== 源码打包规则（必须遵守）==========
# 1. 去掉构建产物：node_modules、.next、web/*/dist、.turbo、*.tsbuildinfo、.DS_Store、.git
# 2. 去掉密钥：*.key、ssl 下 *.crt（ssl/README.md 可保留）
# 3. 去掉 Docker 相关文件：Dockerfile、Dockerfile.*、*.Dockerfile、
#    docker-compose.yml、docker-compose.yaml、docker-compose.*.yml/yaml、
#    compose.yml、compose.yaml、.dockerignore
# 4. 去掉 SQL 文件（*.sql）——交付源码不含数据库迁移/脚本
# 5. 去掉发布产物目录：packages/、images/、*.7z、*.zip、*.tar、*.tar.gz、*.dump、panda-wiki-*-deploy/
# 6. 去掉 arm/ 目录（交付源码不带 ARM 部署目录）
# 7. 去掉仓库根目录 docs/、scripts/（backend/docs、web/**/scripts 保留，编译需要）
# 8. 去掉名为 .env 的文件（任意目录；.env.example 保留）
# 9. 保留 backend/iast/go-active-linux-arm64（插桩构建需要）
# 10. 包根写入 SOURCE_COMMIT；归档名 panda-wiki-src-<commit>.{7z|zip}
# ===========================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${OUT:-}"
FORMAT="7z"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --out) OUT="$2"; shift 2 ;;
    --format) FORMAT="$2"; shift 2 ;;
    -h|--help) sed -n '2,23p' "$0"; exit 0 ;;
    *) echo "未知参数: $1" >&2; exit 1 ;;
  esac
done

case "$FORMAT" in
  7z|zip) ;;
  *) echo "不支持的格式: $FORMAT（仅 7z|zip）" >&2; exit 1 ;;
esac

if [[ -z "$OUT" ]]; then
  if [[ -d /opt/packages ]]; then
    OUT=/opt/packages
  else
    OUT="$ROOT/packages"
  fi
fi
mkdir -p "$OUT"

command -v rsync >/dev/null || { echo "需要 rsync" >&2; exit 1; }
if [[ "$FORMAT" == "7z" ]]; then
  command -v 7z >/dev/null || { echo "需要 7z" >&2; exit 1; }
else
  command -v zip >/dev/null || { echo "需要 zip" >&2; exit 1; }
fi

COMMIT="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || date +%Y%m%d%H%M)"
echo "$COMMIT" > "$ROOT/SOURCE_COMMIT"

NAME="panda-wiki-src-${COMMIT}"
ARCHIVE="${OUT}/${NAME}.${FORMAT}"
STAGE_ROOT="${OUT}/.pack-src-$$"
STAGE="${STAGE_ROOT}/PandaWiki"
rm -rf "$STAGE_ROOT"
mkdir -p "$STAGE"

# 先按规则过滤到暂存目录，再归档（避免排除模式在不同工具下不一致）
rsync -a \
  --exclude '.git/' \
  --exclude 'node_modules/' \
  --exclude '.next/' \
  --exclude 'web/app/dist/' \
  --exclude 'web/admin/dist/' \
  --exclude '.turbo/' \
  --exclude '*.tsbuildinfo' \
  --exclude '.DS_Store' \
  --exclude '*.key' \
  --exclude 'ssl/*.crt' \
  --exclude '**/ssl/*.crt' \
  --exclude 'Dockerfile' \
  --exclude 'Dockerfile.*' \
  --exclude '*.Dockerfile' \
  --exclude 'docker-compose.yml' \
  --exclude 'docker-compose.yaml' \
  --exclude 'docker-compose.*.yml' \
  --exclude 'docker-compose.*.yaml' \
  --exclude 'compose.yml' \
  --exclude 'compose.yaml' \
  --exclude '.dockerignore' \
  --exclude '*.sql' \
  --exclude 'packages/' \
  --exclude 'images/' \
  --exclude '*.7z' \
  --exclude '*.zip' \
  --exclude '*.tar' \
  --exclude '*.tar.gz' \
  --exclude '*.dump' \
  --exclude 'panda-wiki-*-deploy/' \
  --exclude '.pack-src-*/' \
  --exclude '/arm/' \
  --exclude '/docs/' \
  --exclude '/scripts/' \
  --exclude '.env' \
  "$ROOT"/ "$STAGE"/

rm -f "$ARCHIVE"
(
  cd "$STAGE_ROOT"
  if [[ "$FORMAT" == "7z" ]]; then
    7z a -mx5 -bd "$ARCHIVE" PandaWiki >/dev/null
  else
    # -X 去掉 macOS 扩展属性；-r 递归；-q 安静
    zip -qrX "$ARCHIVE" PandaWiki
  fi
)

rm -rf "$STAGE_ROOT"
sha256sum "$ARCHIVE" | tee "${ARCHIVE}.sha256"
ls -lh "$ARCHIVE"

# 抽检：包内不应再出现 Docker 文件 / .sql / arm/ / 根目录 docs、scripts / .env
if [[ "$FORMAT" == "7z" ]]; then
  list_cmd=(7z l "$ARCHIVE")
else
  list_cmd=(zipinfo -1 "$ARCHIVE")
fi
hits="$("${list_cmd[@]}" 2>/dev/null | grep -Ei '(^|/|[[:space:]])Dockerfile(\.|$)|\.sql$|(^|/|[[:space:]])arm/|(^|/|[[:space:]])\.env$|(^|/|[[:space:]])docker-compose[^/]*\.(yml|yaml)$|(^|/|[[:space:]])compose\.(yml|yaml)$|(^|/|[[:space:]])\.dockerignore$|(^|[[:space:]])PandaWiki/(docs|scripts)/' || true)"
if [[ -n "$hits" ]]; then
  echo "ERROR: 包内仍含 Docker 文件、.sql、arm/、docs/、scripts/ 或 .env，请检查排除规则" >&2
  echo "$hits" | head -20 >&2
  exit 1
fi
echo "OK: 已排除 Docker 文件、*.sql、arm/、docs/、scripts/ 与 .env -> $ARCHIVE"
