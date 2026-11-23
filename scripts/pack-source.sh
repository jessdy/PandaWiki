#!/usr/bin/env bash
# 打包交付用源码包（7z）。
#
# 用法：
#   scripts/pack-source.sh                         # 输出到 /opt/packages 或 ./packages
#   scripts/pack-source.sh --out /path/to/dir
#   OUT=/data/packages scripts/pack-source.sh
#
# ========== 源码打包规则（必须遵守）==========
# 1. 去掉构建产物：node_modules、.next、web/*/dist、.turbo、*.tsbuildinfo、.DS_Store、.git
# 2. 去掉密钥：*.key、ssl 下 *.crt（ssl/README.md 可保留）
# 3. 去掉 Dockerfile（Dockerfile、Dockerfile.*、*.Dockerfile）——交付源码不带镜像构建文件
# 4. 去掉 SQL 文件（*.sql）——交付源码不含数据库迁移/脚本
# 5. 去掉发布产物目录：packages/、images/、*.7z、*.tar、*.tar.gz、panda-wiki-*-deploy/
# 6. 保留 backend/iast/go-active-linux-arm64、arm/iast/go-active-linux-arm64（插桩构建需要）
# 7. 包根写入 SOURCE_COMMIT；归档名 panda-wiki-src-<commit>.7z
# ===========================================
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${OUT:-}"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --out) OUT="$2"; shift 2 ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "未知参数: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$OUT" ]]; then
  if [[ -d /opt/packages ]]; then
    OUT=/opt/packages
  else
    OUT="$ROOT/packages"
  fi
fi
mkdir -p "$OUT"

command -v 7z >/dev/null || { echo "需要 7z" >&2; exit 1; }
command -v rsync >/dev/null || { echo "需要 rsync" >&2; exit 1; }

COMMIT="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || date +%Y%m%d%H%M)"
echo "$COMMIT" > "$ROOT/SOURCE_COMMIT"

NAME="panda-wiki-src-${COMMIT}"
ARCHIVE="${OUT}/${NAME}.7z"
STAGE="${OUT}/.pack-src-$$/PandaWiki"
rm -rf "${OUT}/.pack-src-$$"
mkdir -p "$STAGE"

# 先按规则过滤到暂存目录，再 7z（避免 7z 排除模式在不同版本下不一致）
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
  --exclude '*.sql' \
  --exclude 'packages/' \
  --exclude 'images/' \
  --exclude '*.7z' \
  --exclude '*.tar' \
  --exclude '*.tar.gz' \
  --exclude '*.dump' \
  --exclude 'panda-wiki-*-deploy/' \
  --exclude '.pack-src-*/' \
  "$ROOT"/ "$STAGE"/

(
  cd "${OUT}/.pack-src-$$"
  rm -f "$ARCHIVE"
  7z a -mx5 -bd "$ARCHIVE" PandaWiki >/dev/null
)

rm -rf "${OUT}/.pack-src-$$"
sha256sum "$ARCHIVE" | tee "${ARCHIVE}.sha256"
ls -lh "$ARCHIVE"

# 抽检：包内不应再出现 Dockerfile / .sql
hits="$(7z l "$ARCHIVE" | grep -Ei '(^|[[:space:]])Dockerfile(\.|$)|\.sql$' || true)"
if [[ -n "$hits" ]]; then
  echo "ERROR: 包内仍含 Dockerfile 或 .sql，请检查排除规则" >&2
  echo "$hits" | head -20 >&2
  exit 1
fi
echo "OK: 已排除 Dockerfile 与 *.sql -> $ARCHIVE"
