const AUTH_FLAG = 'panda_wiki_authed';
// localStorage 项名（兼容旧版），不是加密密钥；真实 JWT 走 HttpOnly Cookie
const LEGACY_TOKEN_STORAGE_ITEM = 'panda_wiki_token';

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
  return (
    sessionStorage.getItem(AUTH_FLAG) === '1' ||
    !!localStorage.getItem(LEGACY_TOKEN_STORAGE_ITEM)
  );
}

/** 兼容旧版 localStorage token；新登录仅依赖 Cookie */
export function getLegacyBearerToken(): string {
  return localStorage.getItem(LEGACY_TOKEN_STORAGE_ITEM) || '';
}
