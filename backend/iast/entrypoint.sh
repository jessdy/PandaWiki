#!/bin/sh
# 镜像内同时包含普通版与雳鉴 IAST 主动插桩版二进制（*.iast，仅 arm64 构建时产出）。
#
# 开关文件 ${IAST_SWITCH_FILE:-/iast-config/iast.conf}（compose 把宿主机 data/conf/iast/ 只读挂到 /iast-config/）：
#   IAST_ENABLED=true|false
#   IAST_SERVER_IP=雳鉴 Name Server 地址
#   IAST_SERVER_PORT=8888
#   IAST_XSCHEMA_PORTS=443,8443
#   IAST_PROJECT_VERSION=v1.0
# 文件中的值优先于同名环境变量；agent_tag 由 compose 的 IAST_AGENT_TAG 区分 api / consumer。
# 若存在 ${IAST_CONFIG_FILE}（完整 agent toml），开启时直接使用它而不按模板生成。
# 未设置 IAST_ENABLED 时兼容旧方式：存在 toml 或 IAST_SERVER_IP 即开启。
# 开启但配置无效时打印错误并以普通版启动，避免业务中断。
# CMD 通过 ${IAST_SUFFIX} 选择二进制。
set -eu

IAST_SUFFIX=""
export IAST_SUFFIX

SWITCH_FILE="${IAST_SWITCH_FILE:-/iast-config/iast.conf}"
HOME_DIR="${HOME:-/home/panda}"
CFG_DIR="${HOME_DIR}/.moresec_iast_agent"
CFG_FILE="${CFG_DIR}/go-moresec-agent.toml"
TEMPLATE="${IAST_CONFIG_TEMPLATE:-/app/iast/go-moresec-agent.toml.template}"
MOUNTED_CFG="${IAST_CONFIG_FILE:-/iast-config/go-moresec-agent.toml}"
[ -f "${MOUNTED_CFG}" ] || MOUNTED_CFG="/iast-config/go-moresec-agent.toml"

if [ -f "${SWITCH_FILE}" ]; then
  while IFS= read -r line || [ -n "${line}" ]; do
    line="$(printf '%s' "${line}" | tr -d '\r')"
    case "${line}" in ''|'#'*) continue ;; esac
    key="$(printf '%s' "${line%%=*}" | tr -d ' \t')"
    val="$(printf '%s' "${line#*=}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' -e 's/^"\(.*\)"$/\1/' -e "s/^'\(.*\)'\$/\1/")"
    case "${key}" in
      IAST_ENABLED|IAST_SERVER_IP|IAST_SERVER_PORT|IAST_XSCHEMA_PORTS|IAST_PROJECT_VERSION)
        export "${key}=${val}"
        ;;
    esac
  done < "${SWITCH_FILE}"
fi

enabled="$(printf '%s' "${IAST_ENABLED:-}" | tr 'A-Z' 'a-z')"
case "${enabled}" in
  false|0|no|off)
    echo "IAST: 开关为关闭（IAST_ENABLED=${IAST_ENABLED}），以普通版启动"; exec "$@" ;;
  true|1|yes|on) ;;
  "")
    if [ ! -f "${MOUNTED_CFG}" ] && [ -z "${IAST_SERVER_IP:-}" ]; then
      echo "IAST: 未配置，以普通版启动"; exec "$@"
    fi
    ;;
  *)
    echo "ERROR: IAST_ENABLED 取值无效: '${IAST_ENABLED}'（应为 true/false），以普通版启动" >&2; exec "$@" ;;
esac

if ! ls /app/*.iast >/dev/null 2>&1; then
  echo "WARN: 已开启 IAST，但镜像不含插桩版二进制（非 arm64 或构建时缺少 iast_go），以普通版启动" >&2
  exec "$@"
fi

invalid_ip() {
  case "$1" in
    ""|127.0.0.1|localhost|::1|__IAST_SERVER_IP__|改成雳鉴IP|CHANGE_ME) return 0 ;;
  esac
  return 1
}

mkdir -p "${CFG_DIR}"

if [ -f "${MOUNTED_CFG}" ]; then
  ip="$(sed -n 's/^[[:space:]]*IP[[:space:]]*=[[:space:]]*"\([^"]*\)".*/\1/p' "${MOUNTED_CFG}" | head -n1)"
  if invalid_ip "${ip}"; then
    echo "ERROR: ${MOUNTED_CFG} 中雳鉴 IP 无效或仍为占位符: '${ip}'（不可使用 127.0.0.1/localhost），以普通版启动" >&2
    exec "$@"
  fi
  cp "${MOUNTED_CFG}" "${CFG_FILE}"
  echo "IAST: 使用挂载配置 ${MOUNTED_CFG} (IP=${ip})，以插桩版启动"
else
  if invalid_ip "${IAST_SERVER_IP:-}"; then
    echo "ERROR: 已开启 IAST 但 IAST_SERVER_IP 无效或仍为占位符: '${IAST_SERVER_IP:-}'，请编辑 data/conf/iast/iast.conf；以普通版启动" >&2
    exec "$@"
  fi
  if [ ! -f "${TEMPLATE}" ]; then
    echo "ERROR: IAST config template not found: ${TEMPLATE}，以普通版启动" >&2
    exec "$@"
  fi
  sed \
    -e "s|__IAST_SERVER_IP__|${IAST_SERVER_IP}|g" \
    -e "s|__IAST_SERVER_PORT__|${IAST_SERVER_PORT:-8888}|g" \
    -e "s|__IAST_AGENT_TAG__|${IAST_AGENT_TAG:-panda-wiki}|g" \
    -e "s|__IAST_XSCHEMA_PORTS__|${IAST_XSCHEMA_PORTS:-443,8443}|g" \
    -e "s|__IAST_PROJECT_VERSION__|${IAST_PROJECT_VERSION:-v1.0}|g" \
    "${TEMPLATE}" > "${CFG_FILE}"
  echo "IAST: 开关已开启 (server=${IAST_SERVER_IP}:${IAST_SERVER_PORT:-8888}, tag=${IAST_AGENT_TAG:-panda-wiki})，以插桩版启动"
fi

IAST_SUFFIX=".iast"
exec "$@"
