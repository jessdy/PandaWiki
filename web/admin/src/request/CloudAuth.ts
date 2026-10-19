import httpRequest, { ContentType, RequestParams } from './httpClient';

export interface CloudAuthSetting {
  host: string;
  port: number;
}

export const getApiV1SystemCloudAuth = (params: RequestParams = {}) =>
  httpRequest<CloudAuthSetting>({
    path: '/api/v1/system/cloud-auth',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const putApiV1SystemCloudAuth = (
  body: CloudAuthSetting,
  params: RequestParams = {},
) =>
  httpRequest<CloudAuthSetting>({
    path: '/api/v1/system/cloud-auth',
    method: 'PUT',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
