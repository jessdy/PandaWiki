#!/bin/sh
# 雳鉴 Golang 主动插桩：启动前按环境变量生成 ~/.moresec_iast_agent/go-moresec-agent.toml
set -eu

HOME_DIR="${HOME:-/home/panda}"
CFG_DIR="${HOME_DIR}/.moresec_iast_agent"
TEMPLATE="${IAST_CONFIG_TEMPLATE:-/app/iast/go-moresec-agent.toml.template}"

IAST_SERVER_IP="${IAST_SERVER_IP:-}"
IAST_SERVER_PORT="${IAST_SERVER_PORT:-8888}"
IAST_AGENT_TAG="${IAST_AGENT_TAG:-panda-wiki}"
IAST_XSCHEMA_PORTS="${IAST_XSCHEMA_PORTS:-443,8443}"
IAST_PROJECT_VERSION="${IAST_PROJECT_VERSION:-v1.0}"

if [ -z "${IAST_SERVER_IP}" ]; then
  echo "ERROR: IAST_SERVER_IP is required (雳鉴 IAST server IP, 不可使用 127.0.0.1/localhost)" >&2
  exit 1
fi

case "${IAST_SERVER_IP}" in
  127.0.0.1|localhost|::1)
    echo "ERROR: IAST_SERVER_IP 不能为 localhost/127.0.0.1（文档要求页面与上报不可使用本机回环）" >&2
    exit 1
    ;;
esac

if [ ! -f "${TEMPLATE}" ]; then
  echo "ERROR: IAST config template not found: ${TEMPLATE}" >&2
  exit 1
fi

mkdir -p "${CFG_DIR}"
# 使用 | 作分隔符，避免 IP/tag 中特殊字符干扰
sed \
  -e "s|__IAST_SERVER_IP__|${IAST_SERVER_IP}|g" \
  -e "s|__IAST_SERVER_PORT__|${IAST_SERVER_PORT}|g" \
  -e "s|__IAST_AGENT_TAG__|${IAST_AGENT_TAG}|g" \
  -e "s|__IAST_XSCHEMA_PORTS__|${IAST_XSCHEMA_PORTS}|g" \
  -e "s|__IAST_PROJECT_VERSION__|${IAST_PROJECT_VERSION}|g" \
  "${TEMPLATE}" > "${CFG_DIR}/go-moresec-agent.toml"

echo "IAST agent config written: ${CFG_DIR}/go-moresec-agent.toml (server=${IAST_SERVER_IP}:${IAST_SERVER_PORT}, tag=${IAST_AGENT_TAG})"

exec "$@"
