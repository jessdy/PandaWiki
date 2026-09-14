import httpRequest, { ContentType, RequestParams } from './httpClient';

export interface ExpertProfileItem {
  user_id: string;
  account: string;
  role: 'admin' | 'user';
  consult_kb_names: string[];
  name: string;
  introduction: string;
  updated_at?: string;
}

export interface ExpertProfileListResp {
  items: ExpertProfileItem[];
}

export interface UpdateExpertProfileReq {
  name: string;
  introduction: string;
}

export const getApiV1ExpertList = (params: RequestParams = {}) =>
  httpRequest<ExpertProfileListResp>({
    path: '/api/v1/expert/list',
    method: 'GET',
    secure: true,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const putApiV1ExpertProfile = (
  userId: string,
  body: UpdateExpertProfileReq,
  params: RequestParams = {},
) =>
  httpRequest<ExpertProfileItem>({
    path: `/api/v1/expert/${encodeURIComponent(userId)}`,
    method: 'PUT',
    body,
    secure: true,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
