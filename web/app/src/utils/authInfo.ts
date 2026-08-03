import type { GithubComChaitinPandaWikiProApiShareV1AuthInfoResp } from '@/request/pro/types';

/** demo 分支：始终视为已登录 */
export const DEMO_AUTH_INFO: GithubComChaitinPandaWikiProApiShareV1AuthInfoResp =
  {
    id: 1,
    username: '测试用户',
  };

export function isAuthInfoEmpty(
  _authInfo?: GithubComChaitinPandaWikiProApiShareV1AuthInfoResp | null,
): boolean {
  return false;
}
