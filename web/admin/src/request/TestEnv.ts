import httpRequest, { ContentType, RequestParams } from './httpClient';

export interface TestEnvSetting {
  enabled: boolean;
}

export const getApiV1SystemTestEnv = (params: RequestParams = {}) =>
  httpRequest<TestEnvSetting>({
    path: '/api/v1/system/test-env',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const putApiV1SystemTestEnv = (
  body: TestEnvSetting,
  params: RequestParams = {},
) =>
  httpRequest<TestEnvSetting>({
    path: '/api/v1/system/test-env',
    method: 'PUT',
    type: ContentType.Json,
    format: 'json',
    body,
    ...params,
  });
