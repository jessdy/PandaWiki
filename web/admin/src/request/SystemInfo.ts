import httpRequest, { ContentType, RequestParams } from './httpClient';

export interface ClassificationSetting {
  enabled: boolean;
  level: string;
}

export interface SystemPublicInfo {
  classification: ClassificationSetting;
}

export interface SystemAlert {
  type: string;
  level: string;
  message: string;
}

export const getApiV1SystemInfo = (params: RequestParams = {}) =>
  httpRequest<SystemPublicInfo>({
    path: '/api/v1/system/info',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getApiV1SystemAlerts = (params: RequestParams = {}) =>
  httpRequest<SystemAlert[]>({
    path: '/api/v1/system/alerts',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getApiV1SystemClassification = (params: RequestParams = {}) =>
  httpRequest<ClassificationSetting>({
    path: '/api/v1/system/classification',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const putApiV1SystemClassification = (
  body: ClassificationSetting,
  params: RequestParams = {},
) =>
  httpRequest<ClassificationSetting>({
    path: '/api/v1/system/classification',
    method: 'PUT',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
