import httpRequest, { ContentType, RequestParams } from './httpClient';

export interface StandardTermItem {
  id: string;
  term: string;
  aliases: string[];
}

export interface StandardTermsSetting {
  items: StandardTermItem[];
}

export const getApiV1SystemStandardTerms = (
  params: RequestParams = {},
) =>
  httpRequest<StandardTermsSetting>({
    path: '/api/v1/system/standard-terms',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const putApiV1SystemStandardTerms = (
  body: StandardTermsSetting,
  params: RequestParams = {},
) =>
  httpRequest<StandardTermsSetting>({
    path: '/api/v1/system/standard-terms',
    method: 'PUT',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
