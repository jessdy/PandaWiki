import httpRequest, { ContentType, RequestParams } from './httpClient';

export interface BackupInfo {
  name: string;
  size: number;
  created_at: string;
  trigger: string;
  created_by: string;
  schema_version: number;
  tables: number;
  rows: number;
  files: number;
}

export interface BackupSetting {
  enabled: boolean;
  time: string;
  retention: number;
  include_files: boolean;
  alert_percent: number;
  remind_days: number;
}

export interface BackupStorageStatus {
  total_bytes: number;
  free_bytes: number;
  backup_bytes: number;
  used_percent: number;
  alert_percent: number;
  alert: boolean;
  last_success_at?: string;
  remind_days: number;
  remind: boolean;
}

export interface BackupTaskStatus {
  running: boolean;
  kind: string;
  name: string;
  step: string;
  started_at?: string;
  finished_at?: string;
  error: string;
}

export const getApiV1BackupList = (params: RequestParams = {}) =>
  httpRequest<{ list: BackupInfo[] }>({
    path: '/api/v1/backup',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getApiV1BackupStatus = (params: RequestParams = {}) =>
  httpRequest<BackupTaskStatus>({
    path: '/api/v1/backup/status',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getApiV1BackupStorage = (params: RequestParams = {}) =>
  httpRequest<BackupStorageStatus>({
    path: '/api/v1/backup/storage',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getApiV1BackupSetting = (params: RequestParams = {}) =>
  httpRequest<BackupSetting>({
    path: '/api/v1/backup/setting',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const putApiV1BackupSetting = (
  body: BackupSetting,
  params: RequestParams = {},
) =>
  httpRequest<BackupSetting>({
    path: '/api/v1/backup/setting',
    method: 'PUT',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const postApiV1BackupCreate = (params: RequestParams = {}) =>
  httpRequest<BackupInfo>({
    path: '/api/v1/backup',
    method: 'POST',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const postApiV1BackupRestore = (
  body: { name: string; confirm: string },
  params: RequestParams = {},
) =>
  httpRequest<null>({
    path: '/api/v1/backup/restore',
    method: 'POST',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const deleteApiV1Backup = (name: string, params: RequestParams = {}) =>
  httpRequest<null>({
    path: `/api/v1/backup/${encodeURIComponent(name)}`,
    method: 'DELETE',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
