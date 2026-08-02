/**
 * demo 分支：统一拦截 /share/* 请求，返回本地 Mock，不访问后端。
 */
import {
  DEMO_ATTR_COLLECTED_FROM_IMAGE,
  DEMO_ATTR_SPECS,
  DEMO_NODES,
  DEMO_WEB_INFO,
  DEMO_WIDGET_INFO,
  DEMO_WORK_CATEGORY,
  getDemoNodeDetail,
  matchDemoMethods,
} from './fixtures';

/** demo 分支始终开启 Mock */
export const DEMO_MOCK_ENABLED = true;

function parseUrl(input: RequestInfo | URL): URL {
  const raw =
    typeof input === 'string'
      ? input
      : input instanceof URL
        ? input.href
        : input.url;
  try {
    return new URL(raw, 'http://demo.local');
  } catch {
    return new URL('http://demo.local/');
  }
}

function jsonOk(data: unknown, init?: ResponseInit): Response {
  return new Response(JSON.stringify({ success: true, data }), {
    status: 200,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
    ...init,
  });
}

function delay(ms: number) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function readJsonBody(init?: RequestInit): Promise<any> {
  if (!init?.body) return undefined;
  if (typeof init.body === 'string') {
    try {
      return JSON.parse(init.body);
    } catch {
      return undefined;
    }
  }
  return undefined;
}

function chainStep(step: number, title: string, detail: string) {
  return {
    type: 'chain_step',
    content: JSON.stringify({ step, title, detail }),
  };
}

function streamSSE(events: Array<Record<string, unknown>>): Response {
  const encoder = new TextEncoder();
  const stream = new ReadableStream<Uint8Array>({
    async start(controller) {
      for (const event of events) {
        controller.enqueue(
          encoder.encode(`data: ${JSON.stringify(event)}\n\n`),
        );
        const wait =
          event.type === 'data' ? 18 : event.type === 'chain_step' ? 280 : 40;
        await delay(wait);
      }
      controller.close();
    },
  });
  return new Response(stream, {
    status: 200,
    headers: {
      'Content-Type': 'text/event-stream; charset=utf-8',
      'Cache-Control': 'no-cache',
    },
  });
}

/** 附图问答：思维链提取要素 → ATTRIBUTE_PANEL 供用户选择 */
function createImageWorkModeSSE(userMessage?: string): Response {
  const q = (userMessage || '').trim();
  const collected = { ...DEMO_ATTR_COLLECTED_FROM_IMAGE };
  const matched = matchDemoMethods(collected);
  const panelMeta = {
    category: DEMO_WORK_CATEGORY,
    specs: DEMO_ATTR_SPECS,
    collected,
    unrecognized: { 品牌字样: 'DemoCola' } as Record<string, string>,
    methods: matched.methods,
  };
  const fallback =
    `当前为「实战模式」。已识别品类「${DEMO_WORK_CATEGORY}」。` +
    `已采集属性：${Object.entries(collected)
      .map(([k, v]) => `${k}=${v}`)
      .join('、')}。` +
    (matched.methods.length
      ? `可能的开封方法：${matched.methods.map(m => m.name).join('、')}。`
      : '请在面板内补全属性后查找匹配的开封方法。');
  const answer = `<!-- ATTRIBUTE_PANEL ${JSON.stringify(panelMeta)} -->\n${fallback}`;

  const events: Array<Record<string, unknown>> = [
    { type: 'conversation_id', content: `demo-conv-${Date.now()}` },
    { type: 'message_id', content: `demo-msg-${Date.now()}` },
    { type: 'nonce', content: 'demo-nonce' },
    chainStep(
      0,
      '附图检索准备',
      '已收到附图，将依次完成：识别画面 → 品类判断 →（若命中）按品类提取检索要点 → 向量检索。',
    ),
    chainStep(1, '识别图中物体与场景', '正在调用视觉模型…'),
    chainStep(
      1,
      '识别图中物体与场景',
      '画面中央为一只银色圆柱形易拉罐，罐身有蓝红配色标签，顶部可见拉环结构；背景为浅色桌面。' +
        (q ? `用户补充说明：${q}` : ''),
    ),
    chainStep(2, '判断是否属于配置品类', '比对配置中的品类…'),
    chainStep(2, '判断是否属于配置品类', `命中品类：「${DEMO_WORK_CATEGORY}」`),
    chainStep(
      3,
      '按品类提示词提取检索属性',
      '正在结合图片提取与检索相关的属性要点…',
    ),
    chainStep(
      3,
      '按品类提示词提取检索属性',
      '材质偏铝制银色、容量约 330ml、拉环形态尚不明确需人工确认、标签含 DemoCola 字样',
    ),
    chainStep(
      5,
      '实战模式：结构化匹配',
      `品类「${DEMO_WORK_CATEGORY}」已配置规则，本轮命中 ${matched.methods.length} 条。请在面板内调整属性以联动刷新方法卡片。`,
    ),
  ];

  for (const part of answer.split(/(\n)/)) {
    if (part) events.push({ type: 'data', content: part });
  }
  events.push({ type: 'done' });
  return streamSSE(events);
}

function createChatSSE(body?: {
  message?: string;
  image_paths?: string[];
  qa_mode?: string;
}): Response {
  const imagePaths = Array.isArray(body?.image_paths) ? body!.image_paths! : [];
  const hasImages = imagePaths.length > 0;
  // demo 固定实战模式；有附图时走「提取要素 → 属性选择」原流程
  if (hasImages) {
    return createImageWorkModeSSE(body?.message);
  }

  const q = (body?.message || '').trim() || '你的问题';
  const answer = [
    `这是 Demo Mock 回复。`,
    ``,
    `你问的是：「${q}」`,
    ``,
    `当前环境未连接后端，回答由前端本地生成。`,
    ``,
    `上传图片后可体验：附图识别 → 提取要素 → 属性面板选择 → 方法卡片联动。`,
  ].join('\n');

  const events: Array<Record<string, unknown>> = [
    { type: 'conversation_id', content: `demo-conv-${Date.now()}` },
    { type: 'message_id', content: `demo-msg-${Date.now()}` },
    { type: 'nonce', content: 'demo-nonce' },
    {
      type: 'chunk_result',
      chunk_result: {
        node_id: 'doc-welcome',
        name: '欢迎使用 PandaWiki',
        emoji: '👋',
        summary: '了解 Demo 知识库的基本能力与浏览方式',
        node_path_names: ['快速开始', '欢迎使用 PandaWiki'],
      },
    },
    {
      type: 'chunk_result',
      chunk_result: {
        node_id: 'doc-work-mode',
        name: '实战模式说明',
        emoji: '⚡',
        summary: '本 Demo 固定运行在实战模式',
        node_path_names: ['快速开始', '实战模式说明'],
      },
    },
  ];

  for (const part of answer.split(/(\n)/)) {
    if (part) events.push({ type: 'data', content: part });
  }
  events.push({ type: 'done' });
  return streamSSE(events);
}

export async function resolveDemoResponse(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response | null> {
  if (!DEMO_MOCK_ENABLED) return null;

  const url = parseUrl(input);
  const pathname = url.pathname.replace(/\/+$/, '') || '/';
  const method = (init?.method || 'GET').toUpperCase();

  // ---- App ----
  if (pathname.endsWith('/share/v1/app/web/info') && method === 'GET') {
    return jsonOk(DEMO_WEB_INFO);
  }
  if (pathname.endsWith('/share/v1/app/widget/info') && method === 'GET') {
    return jsonOk(DEMO_WIDGET_INFO);
  }

  // ---- Nodes ----
  if (pathname.endsWith('/share/v1/node/list') && method === 'GET') {
    return jsonOk([...DEMO_NODES]);
  }
  if (pathname.endsWith('/share/v1/node/detail') && method === 'GET') {
    const id = url.searchParams.get('id') || '';
    const detail = getDemoNodeDetail(id);
    if (!detail) {
      return new Response(
        JSON.stringify({ success: false, message: '文档不存在', data: null }),
        { status: 404, headers: { 'Content-Type': 'application/json' } },
      );
    }
    return jsonOk(detail);
  }

  // ---- Chat SSE ----
  if (pathname.endsWith('/share/v1/chat/message') && method === 'POST') {
    const body = await readJsonBody(init);
    return createChatSSE(body);
  }

  // ---- Search ----
  if (pathname.endsWith('/share/v1/chat/search') && method === 'POST') {
    const body = await readJsonBody(init);
    const q = String(body?.message || body?.query || '').toLowerCase();
    const matched = DEMO_NODES.filter(
      n =>
        n.type === 2 &&
        (!q ||
          n.name.toLowerCase().includes(q) ||
          n.summary.toLowerCase().includes(q)),
    ).map(n => ({
      node_id: n.id,
      name: n.name,
      emoji: n.emoji,
      summary: n.summary,
      node_path_names: [
        DEMO_NODES.find(p => p.id === n.parent_id)?.name,
        n.name,
      ].filter(Boolean),
    }));
    return jsonOk({ node_result: matched });
  }

  if (pathname.endsWith('/share/v1/chat/widget') && method === 'POST') {
    const body = await readJsonBody(init);
    return createChatSSE(body || { message: 'widget' });
  }
  if (pathname.endsWith('/share/v1/chat/widget/search') && method === 'POST') {
    return jsonOk({ node_result: [] });
  }

  // ---- Conversation ----
  if (pathname.endsWith('/share/v1/conversation/detail') && method === 'GET') {
    return jsonOk({
      id: url.searchParams.get('id') || 'demo-conv',
      subject: 'Demo 会话',
      created_at: new Date().toISOString(),
      messages: [
        {
          role: 'user',
          content: '实战模式是什么？',
          created_at: new Date().toISOString(),
        },
        {
          role: 'assistant',
          content:
            '实战模式面向真实业务问答。本 Demo 固定运行在实战模式，回答为 Mock 数据。',
          created_at: new Date().toISOString(),
        },
      ],
    });
  }

  // ---- Comment / feedback / upload / stat ----
  if (pathname.endsWith('/share/v1/comment/list') && method === 'GET') {
    return jsonOk([]);
  }
  if (pathname.endsWith('/share/v1/comment') && method === 'POST') {
    return jsonOk({ id: `c-${Date.now()}` });
  }
  if (pathname.endsWith('/share/v1/chat/feedback') && method === 'POST') {
    return jsonOk({});
  }
  if (pathname.endsWith('/share/v1/stat/page') && method === 'POST') {
    return jsonOk({});
  }
  if (pathname.endsWith('/share/v1/common/file/upload') && method === 'POST') {
    // 模拟上传耗时，保留原有「先上传再提问」节奏
    await delay(180);
    return jsonOk({ key: `demo-upload-${Date.now()}` });
  }
  if (pathname.endsWith('/share/v1/method_rules/match') && method === 'POST') {
    const body = await readJsonBody(init);
    await delay(120);
    return jsonOk(
      matchDemoMethods(
        (body?.collected && typeof body.collected === 'object'
          ? body.collected
          : {}) as Record<string, string>,
      ),
    );
  }

  // ---- Auth ----
  if (
    pathname.includes('/share/v1/auth/') ||
    pathname.includes('/share/pro/v1/auth/')
  ) {
    if (method === 'GET' && pathname.endsWith('/get')) {
      return jsonOk({
        auth_types: [],
      });
    }
    if (method === 'POST') {
      return jsonOk({
        token: 'demo-token',
        username: 'demo',
      });
    }
    return jsonOk({});
  }

  // ---- Consult (pro) ----
  if (pathname.includes('/share/pro/v1/consult')) {
    if (method === 'GET' && pathname.endsWith('/consult')) {
      return jsonOk({ items: [], total: 0 });
    }
    if (method === 'GET' && /\/consult\/\d+$/.test(pathname)) {
      return jsonOk({
        id: 1,
        title: 'Demo 咨询',
        content: '这是一条 Mock 咨询',
        contact: '',
        attachments: [],
        status: 'pending',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        last_message_at: new Date().toISOString(),
        user_id: 'demo',
        messages: [],
      });
    }
    if (method === 'POST') {
      return jsonOk({
        item: {
          id: Date.now(),
          title: 'Demo 咨询',
          content: '已提交（Mock）',
          contact: '',
          attachments: [],
          status: 'pending',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          last_message_at: new Date().toISOString(),
          user_id: 'demo',
          reply_count: 0,
        },
      });
    }
    return jsonOk({});
  }

  // ---- Captcha ----
  if (pathname.includes('/share/v1/captcha/')) {
    return jsonOk({});
  }

  // 其它 /share 或 /api 请求统一空成功，避免打到后端
  if (
    pathname.includes('/share/') ||
    pathname.includes('/api/v1/') ||
    pathname.includes('/api/pro/')
  ) {
    return jsonOk({});
  }

  return null;
}

export async function demoFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  const mocked = await resolveDemoResponse(input, init);
  if (mocked) return mocked;
  // 非 API：放行（例如静态资源）；demo 下一般不会走到这里
  return fetch(input, init);
}
