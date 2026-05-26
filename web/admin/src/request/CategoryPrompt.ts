import httpRequest, { ContentType, RequestParams } from './httpClient';

/** 单个属性的结构化定义：属性名 + 允许的枚举值列表。 */
export interface CategoryAttributeSpec {
  name: string;
  values?: string[];
}

export interface CategoryPromptItem {
  id: string;
  name: string;
  content: string;
  /** 检索属性维度，逗号分隔。后端写入时由 attribute_specs 派生，旧前端可读。 */
  attributes?: string;
  /** 结构化属性 + 枚举值列表，权威字段。 */
  attribute_specs?: CategoryAttributeSpec[];
}

export const getApiV1CategoryPrompts = (
  query: { id: string },
  params: RequestParams = {},
) =>
  httpRequest<{ items: CategoryPromptItem[] }>({
    path: '/api/v1/knowledge_base/category_prompts',
    method: 'GET',
    query,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const putApiV1CategoryPrompts = (
  body: { kb_id: string; items: CategoryPromptItem[] },
  params: RequestParams = {},
) =>
  httpRequest<unknown>({
    path: '/api/v1/knowledge_base/category_prompts',
    method: 'PUT',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
