/**
 * demo 分支：统一拦截 /share/* 请求，返回本地 Mock，不访问后端。
 * 业务假数据来自 public/demo-fixtures.json（打包后可直接修改该 JSON）。
 */
import {
  buildDemoWebInfo,
  getDemoNodeDetail,
  loadDemoFixtures,
  matchDemoMethods,
  type DemoFixtures,
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

function streamSSE(
  events: Array<Record<string, unknown>>,
  fx: DemoFixtures,
): Response {
  const encoder = new TextEncoder();
  const timing = fx.timing || {
    first_byte_delay_ms: 3000,
    delays: {},
  };
  const delays = timing.delays || {};

  const stream = new ReadableStream<Uint8Array>({
    async start(controller) {
      await delay(timing.first_byte_delay_ms ?? 3000);
      for (const event of events) {
        controller.enqueue(
          encoder.encode(`data: ${JSON.stringify(event)}\n\n`),
        );
        const type = String(event.type || '');
        await delay(delays[type] ?? 120);
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

function pushSlowDataChunks(
  events: Array<Record<string, unknown>>,
  text: string,
  chunkSize = 4,
) {
  for (let i = 0; i < text.length; i += chunkSize) {
    events.push({ type: 'data', content: text.slice(i, i + chunkSize) });
  }
}

/** 附图问答：思维链提取要素 → ATTRIBUTE_PANEL 供用户选择 */
function createImageWorkModeSSE(
  fx: DemoFixtures,
  userMessage?: string,
): Response {
  const q = (userMessage || '').trim();
  const wm = fx.work_mode;
  const collected = { ...(wm.attr_collected_from_image || {}) };
  const matched = matchDemoMethods(fx, collected);
  const panelMeta = {
    category: wm.category,
    specs: wm.attr_specs,
    collected,
    unrecognized: wm.unrecognized || {},
    methods: matched.methods,
  };
  const fallback =
    `当前为「实战模式」。已识别品类「${wm.category}」。` +
    `已采集属性：${Object.entries(collected)
      .map(([k, v]) => `${k}=${v}`)
      .join('、')}。` +
    (matched.methods.length
      ? `可能的开封方法：${matched.methods.map(m => m.name).join('、')}。`
      : '请在面板内补全属性后查找匹配的开封方法。');
  const answer = `<!-- ATTRIBUTE_PANEL ${JSON.stringify(panelMeta)} -->\n${fallback}`;

  const scene =
    wm.image_chain?.scene_detail ||
    '已识别画面主体，正在结合品类配置提取属性。';
  const attrDetail =
    wm.image_chain?.attr_detail ||
    Object.entries(collected)
      .map(([k, v]) => `${k}=${v}`)
      .join('\n');
  const rulesCount =
    wm.image_chain?.rules_count ?? (wm.method_rules?.length || 0);

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
    chainStep(1, '识别图中物体与场景', scene + (q ? `用户补充说明：${q}` : '')),
    chainStep(2, '判断是否属于配置品类', '比对配置中的品类…'),
    chainStep(2, '判断是否属于配置品类', `命中品类：「${wm.category}」`),
    chainStep(
      3,
      '按品类提示词提取检索属性',
      '正在结合图片提取与检索相关的属性要点…',
    ),
    chainStep(3, '按品类提示词提取检索属性', attrDetail),
    chainStep(
      5,
      '实战模式：结构化匹配',
      `品类「${wm.category}」已配置${rulesCount}条规则，本轮命中 ${matched.methods.length} 条。请在面板内调整属性以联动刷新方法卡片。`,
    ),
  ];

  const chunkSize = fx.timing?.data_chunk_size ?? 4;
  const markerEnd = answer.indexOf('-->');
  if (markerEnd >= 0) {
    events.push({ type: 'data', content: answer.slice(0, markerEnd + 3) });
    pushSlowDataChunks(events, answer.slice(markerEnd + 3), chunkSize);
  } else {
    pushSlowDataChunks(events, answer, chunkSize);
  }
  events.push({ type: 'done' });
  return streamSSE(events, fx);
}

function createChatSSE(
  fx: DemoFixtures,
  body?: {
    message?: string;
    image_paths?: string[];
    qa_mode?: string;
  },
): Response {
  const imagePaths = Array.isArray(body?.image_paths) ? body!.image_paths! : [];
  if (imagePaths.length > 0) {
    return createImageWorkModeSSE(fx, body?.message);
  }

  const q = (body?.message || '').trim() || '你的问题';
  const template =
    fx.text_chat?.answer_template ||
    '这是 Demo Mock 回复。\n\n你问的是：「{{message}}」';
  const answer = template.replace(/\{\{message\}\}/g, q);

  const events: Array<Record<string, unknown>> = [
    { type: 'conversation_id', content: `demo-conv-${Date.now()}` },
    { type: 'message_id', content: `demo-msg-${Date.now()}` },
    { type: 'nonce', content: 'demo-nonce' },
  ];

  for (const chunk of fx.text_chat?.chunk_results || []) {
    events.push({ type: 'chunk_result', chunk_result: chunk });
  }

  pushSlowDataChunks(events, answer, fx.timing?.data_chunk_size ?? 3);
  events.push({ type: 'done' });
  return streamSSE(events, fx);
}

export async function resolveDemoResponse(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response | null> {
  if (!DEMO_MOCK_ENABLED) return null;

  const url = parseUrl(input);
  const pathname = url.pathname.replace(/\/+$/, '') || '/';
  const method = (init?.method || 'GET').toUpperCase();

  // 静态 fixtures 本身不走 mock 拦截
  if (pathname.endsWith('/demo-fixtures.json')) {
    return null;
  }

  const fx = await loadDemoFixtures();

  // ---- App ----
  if (pathname.endsWith('/share/v1/app/web/info') && method === 'GET') {
    return jsonOk(buildDemoWebInfo(fx));
  }
  if (pathname.endsWith('/share/v1/app/widget/info') && method === 'GET') {
    return jsonOk(fx.widget_info);
  }

  // ---- Nodes ----
  if (pathname.endsWith('/share/v1/node/list') && method === 'GET') {
    return jsonOk([...(fx.nodes || [])]);
  }
  if (pathname.endsWith('/share/v1/node/detail') && method === 'GET') {
    const id = url.searchParams.get('id') || '';
    const detail = getDemoNodeDetail(fx, id);
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
    return createChatSSE(fx, body);
  }

  // ---- Search ----
  if (pathname.endsWith('/share/v1/chat/search') && method === 'POST') {
    const body = await readJsonBody(init);
    const q = String(body?.message || body?.query || '').toLowerCase();
    const matched = (fx.nodes || [])
      .filter(
        n =>
          n.type === 2 &&
          (!q ||
            String(n.name || '')
              .toLowerCase()
              .includes(q) ||
            String(n.summary || '')
              .toLowerCase()
              .includes(q)),
      )
      .map(n => ({
        node_id: n.id,
        name: n.name,
        emoji: n.emoji,
        summary: n.summary,
        node_path_names: [
          (fx.nodes || []).find(p => p.id === n.parent_id)?.name,
          n.name,
        ].filter(Boolean),
      }));
    return jsonOk({ node_result: matched });
  }

  if (pathname.endsWith('/share/v1/chat/widget') && method === 'POST') {
    const body = await readJsonBody(init);
    return createChatSSE(fx, body || { message: 'widget' });
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
    await delay(180);
    return jsonOk({ key: `demo-upload-${Date.now()}` });
  }
  if (pathname.endsWith('/share/v1/method_rules/match') && method === 'POST') {
    const body = await readJsonBody(init);
    await delay(120);
    return jsonOk(
      matchDemoMethods(
        fx,
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
      return jsonOk({ auth_types: [] });
    }
    if (method === 'POST') {
      return jsonOk({ token: 'demo-token', username: 'demo' });
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

  if (pathname.includes('/share/v1/captcha/')) {
    return jsonOk({});
  }

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
  return fetch(input, init);
}
