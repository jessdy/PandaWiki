const AUTH_FLAG = 'panda_wiki_authed';
const LEGACY_TOKEN_KEY = 'panda_wiki_token';

/** 登录态标记（非 JWT；真实令牌走 HttpOnly Cookie） */
export function markAuthed() {
  sessionStorage.setItem(AUTH_FLAG, '1');
  localStorage.removeItem(LEGACY_TOKEN_KEY);
}

export function clearAuth() {
  sessionStorage.removeItem(AUTH_FLAG);
  localStorage.removeItem(LEGACY_TOKEN_KEY);
}

export function isAuthed(): boolean {
  return (
    sessionStorage.getItem(AUTH_FLAG) === '1' ||
    !!localStorage.getItem(LEGACY_TOKEN_KEY)
  );
}

/** 兼容旧版 localStorage token；新登录仅依赖 Cookie */
export function getLegacyBearerToken(): string {
  return localStorage.getItem(LEGACY_TOKEN_KEY) || '';
}
