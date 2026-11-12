import httpRequest, { ContentType, RequestParams } from './httpClient';

export interface AuditLogItem {
  id: number;
  created_at: string;
  user_id: string;
  account: string;
  role: string;
  ip: string;
  user_agent: string;
  category: string;
  action: string;
  event: string;
  object_type: string;
  object_id: string;
  detail: string;
  method: string;
  path: string;
  result: string;
  status_code: number;
  message: string;
  prev_hash: string;
  hash: string;
}

export interface AuditLogListResp {
  total: number;
  list: AuditLogItem[];
}

export interface AuditLogListQuery {
  page?: number;
  per_page?: number;
  account?: string;
  category?: string;
  result?: string;
  keyword?: string;
  start?: string;
  end?: string;
}

export interface AuditVerifyResult {
  total: number;
  valid: boolean;
  broken_id?: number;
  reason?: string;
  checked_at: string;
}

export interface AuditStorageStatus {
  rows: number;
  table_bytes: number;
  database_bytes: number;
  limit_bytes: number;
  used_percent: number;
  alert_percent: number;
  alert: boolean;
}

export interface AuditStorageSetting {
  limit_mb: number;
  alert_percent: number;
}

export interface AuditExportFile {
  format: string;
  version: number;
  created_at: string;
  exported_by: string;
  count: number;
  kdf: { alg: string; salt: string; iterations: number };
  cipher: { alg: string; nonce: string };
  digest: string;
  data: string;
  signature: string;
}

export const getApiV1AuditLogs = (
  query: AuditLogListQuery = {},
  params: RequestParams = {},
) =>
  httpRequest<AuditLogListResp>({
    path: '/api/v1/audit/logs',
    method: 'GET',
    query,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getApiV1AuditVerify = (params: RequestParams = {}) =>
  httpRequest<AuditVerifyResult>({
    path: '/api/v1/audit/verify',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getApiV1AuditStorage = (params: RequestParams = {}) =>
  httpRequest<AuditStorageStatus>({
    path: '/api/v1/audit/storage',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const putApiV1AuditStorage = (
  body: AuditStorageSetting,
  params: RequestParams = {},
) =>
  httpRequest<AuditStorageSetting>({
    path: '/api/v1/audit/storage',
    method: 'PUT',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const postApiV1AuditExport = (
  body: AuditLogListQuery & { password: string },
  params: RequestParams = {},
) =>
  httpRequest<AuditExportFile>({
    path: '/api/v1/audit/export',
    method: 'POST',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
