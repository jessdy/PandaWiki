/**
 * Demo Mock 数据加载器。
 * 运行时读取 /demo-fixtures.json（打包后位于 public/，可直接改 JSON 无需重编）。
 */

export type DemoMethodRule = {
  id: string;
  name: string;
  description: string;
  node_id: string;
  node_name: string;
  when: Record<string, string>;
};

export type DemoFixtures = {
  kb_id: string;
  timing: {
    first_byte_delay_ms: number;
    delays: Record<string, number>;
    data_chunk_size?: number;
  };
  nodes: any[];
  doc_contents: Record<
    string,
    { name: string; summary: string; content: string }
  >;
  web_info: {
    name: string;
    base_url?: string;
    settings: Record<string, any>;
  };
  widget_info: Record<string, any>;
  work_mode: {
    category: string;
    attr_specs: { name: string; values?: string[] }[];
    attr_collected_from_image: Record<string, string>;
    unrecognized?: Record<string, string>;
    method_rules: DemoMethodRule[];
    image_chain?: {
      scene_detail?: string;
      attr_detail?: string;
      rules_count?: number;
    };
  };
  text_chat?: {
    answer_template?: string;
    chunk_results?: any[];
  };
};

const FIXTURES_URL = '/demo-fixtures.json';
const OPEN_PERM = {
  answerable: 'open' as const,
  visible: 'open' as const,
  visitable: 'open' as const,
};

let cache: DemoFixtures | null = null;
let cacheAt = 0;
/** 客户端短缓存；服务端每次读盘，改 JSON 后刷新即生效 */
const CLIENT_CACHE_MS = 2000;

async function readFixturesFromDisk(): Promise<string> {
  // 用间接 import，避免 webpack 把 node:fs 打进浏览器包导致
  // "Module not found: Can't resolve 'fs/promises'"
  const dynamicImport = new Function('m', 'return import(m)') as (
    m: string,
  ) => Promise<any>;
  const { readFile } = await dynamicImport('node:fs/promises');
  const { join } = await dynamicImport('node:path');
  const candidates = [
    join(process.cwd(), 'public', 'demo-fixtures.json'),
    join(process.cwd(), 'demo-fixtures.json'),
    join(process.cwd(), 'app', 'public', 'demo-fixtures.json'),
  ];
  for (const file of candidates) {
    try {
      return await readFile(file, 'utf8');
    } catch {
      // try next
    }
  }
  throw new Error(
    `demo-fixtures.json not found. Tried:\n${candidates.join('\n')}`,
  );
}

export async function loadDemoFixtures(): Promise<DemoFixtures> {
  const isServer = typeof window === 'undefined';
  if (!isServer && cache && Date.now() - cacheAt < CLIENT_CACHE_MS) {
    return cache;
  }

  let raw: string;
  if (isServer) {
    raw = await readFixturesFromDisk();
  } else {
    const res = await fetch(FIXTURES_URL, { cache: 'no-store' });
    if (!res.ok) {
      throw new Error(`Failed to load ${FIXTURES_URL}: ${res.status}`);
    }
    raw = await res.text();
  }
  const data = JSON.parse(raw) as DemoFixtures;
  cache = data;
  cacheAt = Date.now();
  return data;
}

/** 仅测试/热改 JSON 时清缓存 */
export function clearDemoFixturesCache() {
  cache = null;
}

export function buildDemoWebInfo(fx: DemoFixtures) {
  const recommendDocs = (fx.nodes || [])
    .filter(n => n.type === 2)
    .map(n => ({
      id: n.id,
      name: n.name,
      emoji: n.emoji,
      parent_id: n.parent_id,
      position: n.position,
      summary: n.summary,
      type: n.type,
      permissions: n.permissions || OPEN_PERM,
    }));

  return {
    kb_id: fx.kb_id,
    name: fx.web_info?.name,
    base_url: fx.web_info?.base_url ?? '',
    recommend_nodes: recommendDocs,
    settings: fx.web_info?.settings || {},
  };
}

export function getDemoNodeDetail(fx: DemoFixtures, id: string) {
  const listItem = (fx.nodes || []).find(n => n.id === id);
  if (!listItem) return null;

  if (listItem.type === 1) {
    const children = (fx.nodes || [])
      .filter(n => n.parent_id === id)
      .map(n => ({
        id: n.id,
        name: n.name,
        type: n.type,
        emoji: n.emoji,
        parent_id: n.parent_id,
        position: n.position,
        summary: n.summary,
        status: n.status,
        permissions: n.permissions || OPEN_PERM,
      }));
    return {
      id: listItem.id,
      name: listItem.name,
      type: 1,
      kb_id: fx.kb_id,
      status: 2,
      permissions: listItem.permissions || OPEN_PERM,
      meta: { emoji: listItem.emoji, summary: listItem.summary },
      list: children,
      content: '',
    };
  }

  const doc = fx.doc_contents?.[id];
  return {
    id,
    name: doc?.name || listItem.name,
    type: 2,
    kb_id: fx.kb_id,
    status: 2,
    permissions: listItem.permissions || OPEN_PERM,
    meta: {
      emoji: listItem.emoji,
      summary: doc?.summary || listItem.summary,
      content_type: 'md',
    },
    content: doc?.content || `# ${listItem.name}\n\n暂无内容。`,
  };
}

export function matchDemoMethods(
  fx: DemoFixtures,
  collected: Record<string, string> = {},
) {
  const cleaned = Object.fromEntries(
    Object.entries(collected || {}).filter(
      ([, v]) => typeof v === 'string' && v.trim(),
    ),
  ) as Record<string, string>;

  const rules = fx.work_mode?.method_rules || [];
  const matched = rules
    .filter(rule => {
      const keys = Object.keys(rule.when || {});
      if (keys.length === 0) return true;
      return keys.every(k => cleaned[k] === rule.when[k]);
    })
    .map(({ id, name, description, node_id, node_name }) => ({
      id,
      name,
      description,
      node_id,
      node_name,
    }));

  return {
    category: fx.work_mode?.category || '',
    specs: fx.work_mode?.attr_specs || [],
    methods: matched,
  };
}
