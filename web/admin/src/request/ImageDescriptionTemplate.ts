import httpRequest, { ContentType, RequestParams } from './httpClient';

export interface ImageDescriptionTemplate {
  id: string;
  category: string;
  name: string;
  /** {属性名: 属性值}，前端在选用时按 key 顺序渲染为 K-V 文本。 */
  attributes: Record<string, string>;
  created_at: number;
  updated_at: number;
}

export const getApiV1ImageDescriptionTemplates = (
  query: { kb_id: string; category?: string },
  params: RequestParams = {},
) =>
  httpRequest<{ items: ImageDescriptionTemplate[] }>({
    path: '/api/v1/knowledge_base/image_description_templates',
    method: 'GET',
    query,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const postApiV1ImageDescriptionTemplates = (
  body: {
    kb_id: string;
    category: string;
    name: string;
    attributes: Record<string, string>;
  },
  params: RequestParams = {},
) =>
  httpRequest<{ item: ImageDescriptionTemplate }>({
    path: '/api/v1/knowledge_base/image_description_templates',
    method: 'POST',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
