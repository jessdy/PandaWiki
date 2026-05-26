import httpRequest, { ContentType, RequestParams } from './httpClient';

/**
 * 一条「开封方法规则」。conditions = 属性名 → 允许值数组（OR）。
 * 列出的属性必须全部命中（AND）；未列出的属性视为通配。
 */
export interface MethodRule {
  id: string;
  category: string;
  name: string;
  description?: string;
  conditions: Record<string, string[]>;
  node_id: string;
  created_at?: number;
  updated_at?: number;
}

export const getApiV1MethodRules = (
  query: { kb_id: string; category?: string },
  params: RequestParams = {},
) =>
  httpRequest<{ items: MethodRule[] }>({
    path: '/api/v1/knowledge_base/method_rules',
    method: 'GET',
    query,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const putApiV1MethodRules = (
  body: { kb_id: string; items: MethodRule[] },
  params: RequestParams = {},
) =>
  httpRequest<unknown>({
    path: '/api/v1/knowledge_base/method_rules',
    method: 'PUT',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
