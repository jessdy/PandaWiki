import httpRequest, { ContentType, RequestParams } from './httpClient';

/* ---------------------------------------------------------------- */
/* 类型（与后端 domain.ConsultInquiry / ConsultMessage 对齐）         */
/* ---------------------------------------------------------------- */

export type ConsultAttachmentType = 'image' | 'video';

export interface ConsultAttachment {
  type: ConsultAttachmentType;
  url: string;
  name?: string;
  size?: number;
  mime?: string;
}

export type ConsultStatus = 'pending' | 'processing' | 'replied' | 'closed';

export interface ConsultInquiry {
  id: number;
  user_id: string;
  title: string;
  content: string;
  contact: string;
  attachments: ConsultAttachment[];
  status: ConsultStatus;
  created_at: string;
  updated_at: string;
  last_message_at: string;
}

export interface ConsultInquiryListItem extends ConsultInquiry {
  submitter_name?: string;
  reply_count: number;
}

export interface ConsultMessage {
  id: number;
  inquiry_id: number;
  sender_kind: 'user' | 'admin';
  sender_id: string;
  sender_name: string;
  content: string;
  attachments: ConsultAttachment[];
  created_at: string;
}

export interface ConsultInquiryDetail extends ConsultInquiry {
  messages: ConsultMessage[];
}

export const getApiV1ConsultList = (
  query: {
    status?: ConsultStatus | '';
    keyword?: string;
    page?: number;
    per_page?: number;
  } = {},
  params: RequestParams = {},
) =>
  httpRequest<{ data: ConsultInquiryListItem[]; total: number }>({
    path: '/api/v1/consult/list',
    method: 'GET',
    query,
    secure: true,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getApiV1ConsultOpenCount = (params: RequestParams = {}) =>
  httpRequest<{ count: number }>({
    path: '/api/v1/consult/open_count',
    method: 'GET',
    secure: true,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getApiV1ConsultDetail = (
  query: { id: number },
  params: RequestParams = {},
) =>
  httpRequest<ConsultInquiryDetail>({
    path: '/api/v1/consult/detail',
    method: 'GET',
    query,
    secure: true,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const postApiV1ConsultReply = (
  body: {
    inquiry_id: number;
    content: string;
    attachments?: ConsultAttachment[];
    /** 提交回复后是否同时把单据置为「已回复」状态。 */
    mark_replied?: boolean;
  },
  params: RequestParams = {},
) =>
  httpRequest<{ item: ConsultMessage }>({
    path: '/api/v1/consult/reply',
    method: 'POST',
    body,
    secure: true,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const postApiV1ConsultStatus = (
  body: { inquiry_id: number; status: ConsultStatus },
  params: RequestParams = {},
) =>
  httpRequest<unknown>({
    path: '/api/v1/consult/status',
    method: 'POST',
    body,
    secure: true,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const deleteApiV1Consult = (
  query: { ids: string },
  params: RequestParams = {},
) =>
  httpRequest<unknown>({
    path: '/api/v1/consult',
    method: 'DELETE',
    query,
    secure: true,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
