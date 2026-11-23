import type { NextRequest } from 'next/server';
import { NextResponse } from 'next/server';
import { v4 as uuidv4 } from 'uuid';
import { getShareV1AppWidgetInfo } from './request/ShareApp';

import { parsePathname } from '@/utils';
import { postShareV1StatPage } from '@/request/ShareStat';
import { getShareV1NodeList } from '@/request/ShareNode';
import { getShareV1AppWebInfo } from '@/request/ShareApp';
import { filterEmptyFolders, convertToTree } from '@/utils/drag';
import { deepSearchFirstNode } from '@/utils';

const StatPage = {
  welcome: 1,
  node: 2,
  chat: 3,
  auth: 4,
} as const;

const getFirstNode = async () => {
  const nodeListResult: any = await getShareV1NodeList();
  const tree = filterEmptyFolders(convertToTree(nodeListResult || []));
  return deepSearchFirstNode(tree);
};

const getHomePath = async () => {
  const info = await getShareV1AppWebInfo();
  return info?.settings?.home_page_setting;
};

const homeProxy = async (
  request: NextRequest,
  headers: Record<string, string>,
  session: string,
) => {
  const url = request.nextUrl.clone();
  const { page, id } = parsePathname(url.pathname);
  const requestHeaders = new Headers(request.headers);
  requestHeaders.set('x-current-path', url.pathname);
  requestHeaders.set('x-current-search', url.search);
  const forward = { request: { headers: requestHeaders } };
  try {
    // 获取节点列表
    if (url.pathname === '/') {
      const homePath = await getHomePath();
      if (homePath === 'custom') {
        return NextResponse.rewrite(new URL('/home', request.url), forward);
      } else {
        const [firstNode] = await Promise.all([getFirstNode(), getHomePath()]);
        if (firstNode) {
          return NextResponse.rewrite(
            new URL(`/node/${firstNode.id}`, request.url),
            forward,
          );
        }
        return NextResponse.rewrite(new URL('/node', request.url), forward);
      }
    }

    // 页面上报
    const pages = Object.keys(StatPage);
    if (pages.includes(page) || pages.includes(id)) {
      postShareV1StatPage(
        {
          scene: StatPage[page as keyof typeof StatPage],
          node_id: id || '',
        },
        {
          headers: {
            'x-pw-session-id': session,
            ...headers,
          },
        },
      );
    }

    return NextResponse.next(forward);
  } catch (error) {
    if (
      typeof error === 'object' &&
      error !== null &&
      'message' in error &&
      error.message === 'NEXT_REDIRECT'
    ) {
      return NextResponse.redirect(
        new URL(
          `/auth/login?redirect=${encodeURIComponent(url.pathname + url.search)}`,
          request.url,
        ),
      );
    }
  }

  return NextResponse.next(forward);
};

const proxyShare = async (request: NextRequest) => {
  // 转发到 process.env.TARGET
  const kb_id = request.headers.get('x-kb-id') || process.env.DEV_KB_ID || '';

  const targetOrigin = process.env.TARGET!;
  const targetUrl = new URL(
    request.nextUrl.pathname + request.nextUrl.search,
    targetOrigin,
  );
  // 构造 fetch 选项
  const fetchHeaders = new Headers(request.headers);
  fetchHeaders.set('x-kb-id', kb_id);

  const hasBody = !['GET', 'HEAD'].includes(request.method);
  const fetchOptions: RequestInit = {
    method: request.method,
    headers: fetchHeaders,
    body: hasBody ? request.body : undefined,
    redirect: 'manual',
    ...(hasBody && { duplex: 'half' as const }),
  };
  const proxyRes = await fetch(targetUrl.toString(), fetchOptions);
  // Fetch Headers 合并多条 Set-Cookie 时会丢/并坏；登录会话 Cookie 必须逐条转发
  const headers = new Headers();
  proxyRes.headers.forEach((value, key) => {
    if (key.toLowerCase() === 'set-cookie') return;
    headers.set(key, value);
  });
  const nextRes = new NextResponse(proxyRes.body, {
    status: proxyRes.status,
    statusText: proxyRes.statusText,
    headers,
  });
  const setCookies = collectSetCookies(proxyRes.headers);
  // 页面为 HTTP 时去掉 Secure，避免浏览器静默丢弃会话 Cookie
  const pageIsHTTPS = request.nextUrl.protocol === 'https:';
  for (const cookie of setCookies) {
    nextRes.headers.append(
      'Set-Cookie',
      pageIsHTTPS ? cookie : stripCookieSecure(cookie),
    );
  }
  return nextRes;
};

/** 收集上游全部 Set-Cookie（兼容 getSetCookie / 原始 header） */
function collectSetCookies(h: Headers): string[] {
  if (typeof h.getSetCookie === 'function') {
    const list = h.getSetCookie();
    if (list.length > 0) return list;
  }
  const single = h.get('set-cookie');
  return single ? [single] : [];
}

function stripCookieSecure(raw: string): string {
  return raw.replace(/;\s*Secure/gi, '');
}

export async function proxy(request: NextRequest) {
  const url = request.nextUrl.clone();
  const pathname = url.pathname;
  if (pathname.startsWith('/widget')) {
    const widgetInfo: any = await getShareV1AppWidgetInfo();
    if (widgetInfo) {
      if (!widgetInfo?.settings?.widget_bot_settings?.is_open) {
        return NextResponse.rewrite(new URL('/not-found', request.url));
      }
    }
    return;
  }

  const headers: Record<string, string> = {};
  for (const [key, value] of request.headers.entries()) {
    headers[key] = value;
  }

  let sessionId = request.cookies.get('x-pw-session-id')?.value || '';
  let needSetSessionId = false;

  if (!sessionId) {
    sessionId = uuidv4();
    needSetSessionId = true;
  }

  let response: NextResponse;

  if (pathname.startsWith('/share/')) {
    response = await proxyShare(request);
  } else {
    response = await homeProxy(request, headers, sessionId);
  }

  if (needSetSessionId) {
    response.cookies.set('x-pw-session-id', sessionId, {
      httpOnly: true,
      // 必须与页面协议一致；生产环境若仍是 HTTP，强行 Secure 会被浏览器丢弃
      secure: url.protocol === 'https:',
      sameSite: 'lax',
      path: '/',
    });
  }
  return response;
}

export const config = {
  matcher: [
    '/',
    '/home',
    '/share/:path*',
    '/chat/:path*',
    '/widget',
    '/welcome',
    '/auth/login',
    '/node/:path*',
    '/node',
  ],
};
