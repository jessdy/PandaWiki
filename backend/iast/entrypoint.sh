#!/bin/sh
# 雳鉴 Golang 主动插桩：
# 1) 优先使用卷映射的配置 /iast-config/go-moresec-agent.toml（现场只改宿主机文件）
# 2) 否则按环境变量生成 ~/.moresec_iast_agent/go-moresec-agent.toml
set -eu

HOME_DIR="${HOME:-/home/panda}"
CFG_DIR="${HOME_DIR}/.moresec_iast_agent"
CFG_FILE="${CFG_DIR}/go-moresec-agent.toml"
TEMPLATE="${IAST_CONFIG_TEMPLATE:-/app/iast/go-moresec-agent.toml.template}"
# 宿主机通过 compose 映射到此路径（只读即可）
MOUNTED_CFG="${IAST_MOUNTED_CONFIG:-/iast-config/go-moresec-agent.toml}"

mkdir -p "${CFG_DIR}"

validate_ip() {
  ip="$1"
  case "${ip}" in
    ""|127.0.0.1|localhost|::1|__IAST_SERVER_IP__|改成雳鉴IP|CHANGE_ME)
      echo "ERROR: IAST IP 无效或仍为占位符: '${ip}'（不可使用 127.0.0.1/localhost）" >&2
      return 1
      ;;
  esac
  return 0
}

if [ -f "${MOUNTED_CFG}" ]; then
  # 现场映射方式：复制到 Agent 约定路径
  cp "${MOUNTED_CFG}" "${CFG_FILE}"
  # 粗提取 IP= 行做校验（允许空格）
  mounted_ip="$(sed -n 's/^[[:space:]]*IP[[:space:]]*=[[:space:]]*"\([^"]*\)".*/\1/p' "${CFG_FILE}" | head -n1)"
  if ! validate_ip "${mounted_ip}"; then
    echo "ERROR: 请编辑宿主机映射的 IAST 配置文件，填入真实雳鉴 IP 后再启动" >&2
    exit 1
  fi
  echo "IAST agent config from mounted file: ${MOUNTED_CFG} -> ${CFG_FILE} (IP=${mounted_ip})"
  exec "$@"
fi

# ---------- 环境变量方式（兼容原流程）----------
IAST_SERVER_IP="${IAST_SERVER_IP:-}"
IAST_SERVER_PORT="${IAST_SERVER_PORT:-8888}"
IAST_AGENT_TAG="${IAST_AGENT_TAG:-panda-wiki}"
IAST_XSCHEMA_PORTS="${IAST_XSCHEMA_PORTS:-443,8443}"
IAST_PROJECT_VERSION="${IAST_PROJECT_VERSION:-v1.0}"

if [ -z "${IAST_SERVER_IP}" ]; then
  echo "ERROR: 未映射 /iast-config/go-moresec-agent.toml，且未设置 IAST_SERVER_IP" >&2
  echo "现场推荐：在宿主机编辑 data/conf/iast/*.toml 并映射进容器" >&2
  exit 1
fi

if ! validate_ip "${IAST_SERVER_IP}"; then
  exit 1
fi

if [ ! -f "${TEMPLATE}" ]; then
  echo "ERROR: IAST config template not found: ${TEMPLATE}" >&2
  exit 1
fi

sed \
  -e "s|__IAST_SERVER_IP__|${IAST_SERVER_IP}|g" \
  -e "s|__IAST_SERVER_PORT__|${IAST_SERVER_PORT}|g" \
  -e "s|__IAST_AGENT_TAG__|${IAST_AGENT_TAG}|g" \
  -e "s|__IAST_XSCHEMA_PORTS__|${IAST_XSCHEMA_PORTS}|g" \
  -e "s|__IAST_PROJECT_VERSION__|${IAST_PROJECT_VERSION}|g" \
  "${TEMPLATE}" > "${CFG_FILE}"

echo "IAST agent config written: ${CFG_FILE} (server=${IAST_SERVER_IP}:${IAST_SERVER_PORT}, tag=${IAST_AGENT_TAG})"

exec "$@"
