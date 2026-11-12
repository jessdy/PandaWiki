import httpRequest, { ContentType, RequestParams } from './httpClient';

export interface UserApprovalItem {
  id: number;
  type: string;
  status: string;
  summary: string;
  target_user_id: string;
  target_account: string;
  kb_id: string;
  requester_id: string;
  requester_account: string;
  reviewer_id: string;
  reviewer_account: string;
  review_comment: string;
  error: string;
  created_at: string;
  reviewed_at?: string;
}

export interface ApprovalListResp {
  total: number;
  list: UserApprovalItem[];
}

export const getApiV1ApprovalList = (
  query: { status?: string; page?: number; per_page?: number } = {},
  params: RequestParams = {},
) =>
  httpRequest<ApprovalListResp>({
    path: '/api/v1/approval',
    method: 'GET',
    query,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getApiV1ApprovalPendingCount = (params: RequestParams = {}) =>
  httpRequest<{ count: number }>({
    path: '/api/v1/approval/pending_count',
    method: 'GET',
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const postApiV1ApprovalApprove = (
  id: number,
  body: { comment?: string } = {},
  params: RequestParams = {},
) =>
  httpRequest<null>({
    path: `/api/v1/approval/${id}/approve`,
    method: 'POST',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const postApiV1ApprovalReject = (
  id: number,
  body: { comment?: string } = {},
  params: RequestParams = {},
) =>
  httpRequest<null>({
    path: `/api/v1/approval/${id}/reject`,
    method: 'POST',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
