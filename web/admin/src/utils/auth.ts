const AUTH_FLAG = 'panda_wiki_authed';
// localStorage 项名（兼容旧版），不是加密密钥；真实 JWT 走 HttpOnly Cookie
const LEGACY_TOKEN_STORAGE_ITEM = 'panda_wiki_token';
const LOGIN_PORTAL_KEY = 'panda_wiki_login_portal';

export type LoginPortal = 'admin' | 'security' | 'audit';

/** 登录态标记（非 JWT；真实令牌走 HttpOnly Cookie） */
export function markAuthed() {
  sessionStorage.setItem(AUTH_FLAG, '1');
  localStorage.removeItem(LEGACY_TOKEN_STORAGE_ITEM);
}

export function clearAuth() {
  sessionStorage.removeItem(AUTH_FLAG);
  localStorage.removeItem(LEGACY_TOKEN_STORAGE_ITEM);
}

export function isAuthed(): boolean {
  return sessionStorage.getItem(AUTH_FLAG) === '1';
}

/** 不再从 localStorage 读取 JWT。保留函数是为了现有调用点；登录时会清掉旧值。 */
export function getLegacyBearerToken(): string {
  return '';
}

export function setLoginPortal(portal: LoginPortal) {
  sessionStorage.setItem(LOGIN_PORTAL_KEY, portal);
}

export function getLoginPortal(): LoginPortal {
  const v = sessionStorage.getItem(LOGIN_PORTAL_KEY);
  if (v === 'security' || v === 'audit' || v === 'admin') return v;
  return 'admin';
}

/** 按当前会话入口返回对应登录页路径 */
export function loginPathForPortal(portal?: LoginPortal): string {
  const p = portal || getLoginPortal();
  if (p === 'security') return '/login/security';
  if (p === 'audit') return '/login/audit';
  return '/login';
}

export function clearLoginPortal() {
  sessionStorage.removeItem(LOGIN_PORTAL_KEY);
}
