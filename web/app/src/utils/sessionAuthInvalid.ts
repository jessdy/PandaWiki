/**
 * 判断接口错误是否由「后端 session 失效」引起。
 *
 * 现状：httpClient 对 200 + code!=0 业务错误统一 Promise.reject({ code, message, ... })，
 * 401 才会走 redirect 流程。后端在 session 失效时返回 "请先登录" 这类文案，
 * 无独立错误码，因此只能从 message 关键字反推。
 *
 * 命中策略尽量保守（避免误清登录态）：
 *   - HTTP 401（少数场景，httpClient 已自动跳登录）
 *   - 业务 message 严格匹配以下短语之一
 */
const UNAUTH_MESSAGE_PATTERNS = ['请先登录', '未登录', '请登录后'];

export function isSessionAuthInvalidError(e: unknown): boolean {
  if (e == null || typeof e !== 'object') return false;
  const err = e as { code?: number; message?: string; status?: number };
  if (err.code === 401 || err.status === 401) return true;
  const msg = (err.message || '').trim();
  if (!msg) return false;
  return UNAUTH_MESSAGE_PATTERNS.some(p => msg.includes(p));
}
