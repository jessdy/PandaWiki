'use client';

/** demo 分支：跳过登录校验，直接放行 */
export default function AuthGuard({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
