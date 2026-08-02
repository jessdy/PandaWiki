/** demo 分支本地假数据，不依赖后端 */

export const DEMO_KB_ID = 'demo-kb-001';

const openPerm = {
  answerable: 'open' as const,
  visible: 'open' as const,
  visitable: 'open' as const,
};

export const DEMO_NODES = [
  {
    id: 'folder-getting-started',
    name: '快速开始',
    type: 1,
    emoji: '📁',
    parent_id: '',
    position: 1,
    status: 2,
    summary: '入门指南与环境说明',
    permissions: openPerm,
  },
  {
    id: 'doc-welcome',
    name: '欢迎使用 PandaWiki',
    type: 2,
    emoji: '👋',
    parent_id: 'folder-getting-started',
    position: 1,
    status: 2,
    summary: '了解 Demo 知识库的基本能力与浏览方式',
    permissions: openPerm,
  },
  {
    id: 'doc-work-mode',
    name: '实战模式说明',
    type: 2,
    emoji: '⚡',
    parent_id: 'folder-getting-started',
    position: 2,
    status: 2,
    summary: '本 Demo 固定运行在实战模式，用于产品演示',
    permissions: openPerm,
  },
  {
    id: 'folder-product',
    name: '产品文档',
    type: 1,
    emoji: '📦',
    parent_id: '',
    position: 2,
    status: 2,
    summary: '产品功能与使用指南',
    permissions: openPerm,
  },
  {
    id: 'doc-faq',
    name: '常见问题',
    type: 2,
    emoji: '❓',
    parent_id: 'folder-product',
    position: 1,
    status: 2,
    summary: '高频问题与排查建议',
    permissions: openPerm,
  },
  {
    id: 'doc-api',
    name: '接口概览',
    type: 2,
    emoji: '🔌',
    parent_id: 'folder-product',
    position: 2,
    status: 2,
    summary: '对外常用接口与调用约定',
    permissions: openPerm,
  },
] as const;

const DOC_CONTENTS: Record<
  string,
  { name: string; summary: string; content: string }
> = {
  'doc-welcome': {
    name: '欢迎使用 PandaWiki',
    summary: '了解 Demo 知识库的基本能力与浏览方式',
    content: `# 欢迎使用 PandaWiki

这是一个 **纯前端 Demo**：所有数据均为本地 Mock，无需启动后端。

## 你可以做什么

- 浏览左侧目录与文档
- 使用智能问答（流式回复为 Mock）
- 体验固定的「实战模式」界面

## 说明

当前分支已关闭登录校验，并以演示用户身份访问。
`,
  },
  'doc-work-mode': {
    name: '实战模式说明',
    summary: '本 Demo 固定运行在实战模式，用于产品演示',
    content: `# 实战模式说明

本 Demo 默认并锁定为 **实战模式**（\`work\`），界面无法切换到培训模式。

实战模式面向真实业务问答场景，强调结构化属性识别与检索范围控制。
`,
  },
  'doc-faq': {
    name: '常见问题',
    summary: '高频问题与排查建议',
    content: `# 常见问题

## 为什么看不到后端请求？

Demo 分支在 HTTP 客户端层拦截了 \`/share/*\` 请求，并返回本地假数据。

## 问答为什么是固定回复？

\`/share/v1/chat/message\` 的 SSE 流由前端 Mock，便于离线演示。
`,
  },
  'doc-api': {
    name: '接口概览',
    summary: '对外常用接口与调用约定',
    content: `# 接口概览

| 接口 | 说明 |
| --- | --- |
| \`GET /share/v1/app/web/info\` | 站点配置 |
| \`GET /share/v1/node/list\` | 文档目录 |
| \`GET /share/v1/node/detail\` | 文档详情 |
| \`POST /share/v1/chat/message\` | 智能问答（SSE） |
`,
  },
};

export function getDemoNodeDetail(id: string) {
  const listItem = DEMO_NODES.find(n => n.id === id);
  if (!listItem) return null;

  if (listItem.type === 1) {
    const children = DEMO_NODES.filter(n => n.parent_id === id).map(n => ({
      id: n.id,
      name: n.name,
      type: n.type,
      emoji: n.emoji,
      parent_id: n.parent_id,
      position: n.position,
      summary: n.summary,
      status: n.status,
      permissions: openPerm,
    }));
    return {
      id: listItem.id,
      name: listItem.name,
      type: 1,
      kb_id: DEMO_KB_ID,
      status: 2,
      permissions: openPerm,
      meta: { emoji: listItem.emoji, summary: listItem.summary },
      list: children,
      content: '',
    };
  }

  const doc = DOC_CONTENTS[id];
  return {
    id,
    name: doc?.name || listItem.name,
    type: 2,
    kb_id: DEMO_KB_ID,
    status: 2,
    permissions: openPerm,
    meta: {
      emoji: listItem.emoji,
      summary: doc?.summary || listItem.summary,
      content_type: 'md',
    },
    content: doc?.content || `# ${listItem.name}\n\n暂无内容。`,
  };
}

const recommendDocs = DEMO_NODES.filter(n => n.type === 2).map(n => ({
  id: n.id,
  name: n.name,
  emoji: n.emoji,
  parent_id: n.parent_id,
  position: n.position,
  summary: n.summary,
  type: n.type,
  permissions: openPerm,
}));

export const DEMO_WEB_INFO = {
  kb_id: DEMO_KB_ID,
  name: 'PandaWiki Demo',
  base_url: '',
  recommend_nodes: recommendDocs,
  settings: {
    title: 'PandaWiki Demo',
    desc: '纯前端演示站点（Mock 数据）',
    keyword: 'PandaWiki,Demo',
    theme_mode: 'light',
    home_page_setting: 'custom',
    icon: '',
    catalog_settings: {
      catalog_folder: 1,
      catalog_visible: 1,
      catalog_width: 280,
    },
    web_app_custom_style: {
      header_search_placeholder: '搜索文档或提问…',
    },
    web_app_landing_configs: [
      {
        type: 'banner',
        banner_config: {
          title: 'PandaWiki Demo',
          subtitle: '无需后端，浏览文档并体验实战模式问答',
          title_font_size: 48,
          subtitle_font_size: 18,
          placeholder: '试试问：实战模式是什么？',
          hot_search: [
            '实战模式是什么',
            '如何浏览文档',
            '常见问题有哪些',
            '接口概览',
          ],
          btns: [],
        },
      },
      {
        type: 'basic_doc',
        basic_doc_config: {
          title: '推荐文档',
        },
        node_ids: recommendDocs.map(d => d.id),
        nodes: recommendDocs,
      },
      {
        type: 'faq',
        faq_config: {
          title: '快捷入口',
          list: [
            {
              question: '欢迎页',
              link: '/node/doc-welcome',
            },
            {
              question: '实战模式说明',
              link: '/node/doc-work-mode',
            },
            {
              question: '常见问题',
              link: '/node/doc-faq',
            },
          ],
        },
      },
    ],
    recommend_questions: [
      '实战模式是什么？',
      '这个 Demo 需要后端吗？',
      '如何查看文档目录？',
    ],
  },
};

export const DEMO_WIDGET_INFO = {
  name: 'PandaWiki Demo Widget',
  settings: {
    widget_bot_settings: {
      is_open: false,
    },
  },
};

/** 实战模式附图识别：品类 / 属性枚举 / 开封方法规则（Mock） */
export const DEMO_WORK_CATEGORY = '易拉罐';

export const DEMO_ATTR_SPECS = [
  { name: '材质', values: ['铝', '马口铁'] },
  { name: '尺寸', values: ['250ml', '330ml', '500ml'] },
  { name: '拉环类型', values: ['普通拉环', '全开盖'] },
];

/** AI 从附图「识别」出的初始属性（部分识别，留给用户补全） */
export const DEMO_ATTR_COLLECTED_FROM_IMAGE: Record<string, string> = {
  材质: '铝',
  尺寸: '330ml',
};

export type DemoMethodRule = {
  id: string;
  name: string;
  description: string;
  node_id: string;
  node_name: string;
  /** 规则条件：collected 需包含这些键值才命中；空对象表示始终命中 */
  when: Record<string, string>;
};

export const DEMO_METHOD_RULES: DemoMethodRule[] = [
  {
    id: 'rule-alu-std',
    name: '铝罐标准开封',
    description: '适用于铝制普通拉环易拉罐，沿拉环方向平稳拉开。',
    node_id: 'doc-faq',
    node_name: '常见问题',
    when: { 材质: '铝', 拉环类型: '普通拉环' },
  },
  {
    id: 'rule-alu-full',
    name: '铝罐全开盖开封',
    description: '全开盖设计，拉环拉开后盖板整体掀开。',
    node_id: 'doc-work-mode',
    node_name: '实战模式说明',
    when: { 材质: '铝', 拉环类型: '全开盖' },
  },
  {
    id: 'rule-tin',
    name: '马口铁罐开封',
    description: '马口铁罐体较硬，开封时注意拉环受力点。',
    node_id: 'doc-api',
    node_name: '接口概览',
    when: { 材质: '马口铁' },
  },
  {
    id: 'rule-generic',
    name: '通用开封指引',
    description: '属性尚未完全匹配专用规则时的兜底方法。',
    node_id: 'doc-welcome',
    node_name: '欢迎使用 PandaWiki',
    when: {},
  },
];

/** 按已采集属性过滤命中方法（与后台 MatchesCollected 语义接近） */
export function matchDemoMethods(collected: Record<string, string> = {}) {
  const cleaned = Object.fromEntries(
    Object.entries(collected || {}).filter(
      ([, v]) => typeof v === 'string' && v.trim(),
    ),
  ) as Record<string, string>;

  const matched = DEMO_METHOD_RULES.filter(rule => {
    const keys = Object.keys(rule.when);
    if (keys.length === 0) return true;
    return keys.every(k => cleaned[k] === rule.when[k]);
  }).map(({ id, name, description, node_id, node_name }) => ({
    id,
    name,
    description,
    node_id,
    node_name,
  }));

  return {
    category: DEMO_WORK_CATEGORY,
    specs: DEMO_ATTR_SPECS,
    methods: matched,
  };
}
