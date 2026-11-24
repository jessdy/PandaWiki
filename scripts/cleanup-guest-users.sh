#!/usr/bin/env bash
# 在 106 上按 users.id 清理访客账号。
#
# 只删除 role=guest 的用户。任一 ID 不存在或不是访客时，整批回滚，不删任何数据。
# 默认只预览；加上 --apply 才真正提交。
#
# 用法（在 106 上，部署目录默认 /opt/panda-wiki）：
#   scripts/cleanup-guest-users.sh <users.id> [users.id ...]
#   scripts/cleanup-guest-users.sh --file ids.txt
#   scripts/cleanup-guest-users.sh --apply <users.id> ...
#
# ids.txt 每行一个 users.id，空行和 # 开头的行会忽略。
# 这是 users 表的 UUID，不是 TGT 平台用户号。
#
# 会删除：users、kb_users、auth_groups.user_ids、auths、
#         这些登录会话下的咨询单/评论/文档反馈、api_tokens、
#         以该用户为对象的待审批单、expert_profiles。
# 不删：审计日志、文档节点。
set -euo pipefail

ENV_FILE="${ENV_FILE:-/opt/panda-wiki/.env}"
PG_CONTAINER="${PG_CONTAINER:-panda-wiki-postgres}"
PGUSER="${PGUSER:-panda-wiki}"
PGDATABASE="${PGDATABASE:-panda-wiki}"

APPLY=0
FILE=""
IDS=()

usage() {
  sed -n '2,18p' "$0"
  exit 2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --apply) APPLY=1; shift ;;
    --dry-run) APPLY=0; shift ;;
    --file)
      [[ $# -ge 2 ]] || usage
      FILE="$2"
      shift 2
      ;;
    -h|--help) usage ;;
    --) shift; break ;;
    -*) echo "未知参数: $1" >&2; usage ;;
    *) IDS+=("$1"); shift ;;
  esac
done
if [[ $# -gt 0 ]]; then
  IDS+=("$@")
fi

if [[ -n "$FILE" ]]; then
  [[ -f "$FILE" ]] || { echo "找不到文件: $FILE" >&2; exit 2; }
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%%#*}"
    line="$(printf '%s' "$line" | tr -d '[:space:]')"
    [[ -n "$line" ]] && IDS+=("$line")
  done < "$FILE"
fi

if [[ ${#IDS[@]} -eq 0 ]]; then
  echo "请至少指定一个 users.id" >&2
  usage
fi

uuid_re='^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'
VALUES=""
SEEN=" "
for id in "${IDS[@]}"; do
  if [[ ! "$id" =~ $uuid_re ]]; then
    echo "不是 users.id（UUID）: $id" >&2
    exit 2
  fi
  id="$(printf '%s' "$id" | tr '[:upper:]' '[:lower:]')"
  if [[ "$SEEN" == *" $id "* ]]; then
    continue
  fi
  SEEN+="$id "
  if [[ -n "$VALUES" ]]; then
    VALUES+=","
  fi
  VALUES+="('$id')"
done

PGPASSWORD_VALUE="${POSTGRES_PASSWORD:-}"
if [[ -z "$PGPASSWORD_VALUE" && -f "$ENV_FILE" ]]; then
  PGPASSWORD_VALUE="$(grep -E '^POSTGRES_PASSWORD=' "$ENV_FILE" | head -1 | cut -d= -f2- | sed -e 's/^["'\'']//' -e 's/["'\'']$//')"
fi

PSQL=(psql -U "$PGUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -X)
if docker inspect "$PG_CONTAINER" >/dev/null 2>&1; then
  run_sql() {
    docker exec -i -e "PGPASSWORD=${PGPASSWORD_VALUE}" "$PG_CONTAINER" "${PSQL[@]}"
  }
else
  export PGPASSWORD="$PGPASSWORD_VALUE"
  export PGHOST="${PGHOST:-127.0.0.1}"
  export PGPORT="${PGPORT:-5434}"
  run_sql() { "${PSQL[@]}"; }
fi

ENDING="ROLLBACK;"
MODE="预览（未提交）"
if [[ "$APPLY" -eq 1 ]]; then
  ENDING="COMMIT;"
  MODE="已提交"
fi

echo "模式: $MODE"
echo "目标 ID 数: $(printf '%s' "$SEEN" | wc -w | tr -d ' ')"

run_sql <<SQL
BEGIN;

CREATE TEMP TABLE req_ids(id text PRIMARY KEY) ON COMMIT DROP;
INSERT INTO req_ids(id) VALUES ${VALUES};

\\echo
\\echo '---- 核对 ----'
SELECT r.id,
       COALESCE(u.account, '') AS account,
       COALESCE(u.role, '') AS role,
       CASE
         WHEN u.id IS NULL THEN '不存在'
         WHEN u.role <> 'guest' THEN '不是访客'
         ELSE '可删除'
       END AS check_result
FROM req_ids r
LEFT JOIN users u ON u.id = r.id
ORDER BY r.id;

DO \$\$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM req_ids r
    LEFT JOIN users u ON u.id = r.id
    WHERE u.id IS NULL OR u.role <> 'guest'
  ) THEN
    RAISE EXCEPTION '存在无效 ID 或非 guest 用户，已全部回滚';
  END IF;
END
\$\$;

CREATE TEMP TABLE guest_rows ON COMMIT DROP AS
SELECT id, account FROM users
WHERE id IN (SELECT id FROM req_ids) AND role = 'guest';

CREATE TEMP TABLE auth_rows ON COMMIT DROP AS
SELECT a.id, a.union_id, a.kb_id
FROM auths a
JOIN guest_rows g ON a.union_id = g.account;

\\echo
\\echo '---- 关联登录会话 ----'
SELECT id AS auth_id, union_id AS account, kb_id FROM auth_rows ORDER BY id;

DELETE FROM kb_users WHERE user_id IN (SELECT id FROM guest_rows);

UPDATE auth_groups ag
SET user_ids = (
  SELECT COALESCE(array_agg(x), '{}')
  FROM unnest(COALESCE(ag.user_ids, '{}')) AS x
  WHERE x NOT IN (SELECT id FROM guest_rows)
)
WHERE EXISTS (
  SELECT 1 FROM unnest(COALESCE(ag.user_ids, '{}')) AS x
  WHERE x IN (SELECT id FROM guest_rows)
);

DO \$\$
BEGIN
  IF to_regclass('public.consult_inquiries') IS NOT NULL THEN
    DELETE FROM consult_inquiries
    WHERE user_id IN (SELECT id::text FROM auth_rows);
  END IF;
  IF to_regclass('public.document_feedbacks') IS NOT NULL THEN
    DELETE FROM document_feedbacks
    WHERE user_id IN (SELECT id::text FROM auth_rows);
  END IF;
  IF to_regclass('public.comments') IS NOT NULL THEN
    DELETE FROM comments
    WHERE COALESCE(info->>'auth_user_id', '') <> ''
      AND COALESCE(info->>'auth_user_id', '') <> '0'
      AND info->>'auth_user_id' IN (SELECT id::text FROM auth_rows);
  END IF;
  IF to_regclass('public.api_tokens') IS NOT NULL THEN
    DELETE FROM api_tokens WHERE user_id IN (SELECT id FROM guest_rows);
  END IF;
  IF to_regclass('public.user_approvals') IS NOT NULL THEN
    DELETE FROM user_approvals WHERE target_user_id IN (SELECT id FROM guest_rows);
  END IF;
  IF to_regclass('public.expert_profiles') IS NOT NULL THEN
    DELETE FROM expert_profiles WHERE user_id IN (SELECT id FROM guest_rows);
  END IF;
END
\$\$;

DELETE FROM auths WHERE id IN (SELECT id FROM auth_rows);
DELETE FROM users WHERE id IN (SELECT id FROM guest_rows) AND role = 'guest';

\\echo
\\echo '---- 删除后剩余（应为空） ----'
SELECT id, account, role FROM users WHERE id IN (SELECT id FROM req_ids);

${ENDING}
SQL

echo "完成: $MODE"
