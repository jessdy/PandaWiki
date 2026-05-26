/**
 * 与后端 backend/usecase/chat.go 中 `attributePanelMeta` 结构对齐。
 * 当后端识别出品类且该品类配置了 method_rules 时，状态机不再走 RAG，
 * 改为推送 `<!-- ATTRIBUTE_PANEL {...} -->\n人类可读 fallback` 形式的 assistant message。
 */

export interface AttrPanelSpec {
  name: string;
  values?: string[];
}

export interface AttrPanelMethod {
  id: string;
  name: string;
  description?: string;
  node_id: string;
  node_name?: string;
}

export interface AttributePanelMeta {
  category: string;
  specs: AttrPanelSpec[];
  collected?: Record<string, string>;
  unrecognized?: Record<string, string>;
  methods: AttrPanelMethod[];
}

const ATTRIBUTE_PANEL_REGEX =
  /<!--\s*ATTRIBUTE_PANEL\s+(\{[\s\S]*?\})\s*-->\s*\n?/;

export function extractAttributePanel(content: string): {
  meta: AttributePanelMeta | null;
  text: string;
} {
  if (!content) return { meta: null, text: '' };
  const match = content.match(ATTRIBUTE_PANEL_REGEX);
  if (!match) return { meta: null, text: content };
  try {
    const parsed = JSON.parse(match[1]);
    const specs: AttrPanelSpec[] = Array.isArray(parsed?.specs)
      ? parsed.specs
          .filter(
            (s: unknown) => s && typeof (s as AttrPanelSpec).name === 'string',
          )
          .map((s: AttrPanelSpec) => ({
            name: String(s.name),
            values: Array.isArray(s.values)
              ? s.values.filter(v => typeof v === 'string')
              : [],
          }))
      : [];
    const methods: AttrPanelMethod[] = Array.isArray(parsed?.methods)
      ? parsed.methods
          .filter(
            (m: unknown) => m && typeof (m as AttrPanelMethod).id === 'string',
          )
          .map((m: AttrPanelMethod) => ({
            id: String(m.id),
            name: String(m.name || ''),
            description:
              typeof m.description === 'string' ? m.description : undefined,
            node_id: String(m.node_id || ''),
            node_name:
              typeof m.node_name === 'string' ? m.node_name : undefined,
          }))
      : [];
    const meta: AttributePanelMeta = {
      category: typeof parsed.category === 'string' ? parsed.category : '',
      specs,
      collected:
        parsed.collected && typeof parsed.collected === 'object'
          ? (Object.fromEntries(
              Object.entries(parsed.collected).filter(
                ([k, v]) => typeof k === 'string' && typeof v === 'string',
              ),
            ) as Record<string, string>)
          : undefined,
      unrecognized:
        parsed.unrecognized && typeof parsed.unrecognized === 'object'
          ? (Object.fromEntries(
              Object.entries(parsed.unrecognized).filter(
                ([k, v]) => typeof k === 'string' && typeof v === 'string',
              ),
            ) as Record<string, string>)
          : undefined,
      methods,
    };
    return { meta, text: content.replace(match[0], '') };
  } catch {
    return { meta: null, text: content.replace(match[0], '') };
  }
}
