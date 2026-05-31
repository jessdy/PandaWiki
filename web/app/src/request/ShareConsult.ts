import httpRequest, {
  ContentType,
  DomainResponse,
  RequestParams,
} from './httpClient';

/* ---------------------------------------------------------------- */
/* 类型                                                              */
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
  /** 后续追加的消息条数（不含首条提问） */
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

/* ---------------------------------------------------------------- */
/* API                                                               */
/* ---------------------------------------------------------------- */

export const postShareV1ConsultCreate = (
  body: {
    content: string;
    contact?: string;
    attachments?: ConsultAttachment[];
  },
  params: RequestParams = {},
) =>
  httpRequest<DomainResponse & { data?: { item: ConsultInquiry } }>({
    path: '/share/pro/v1/consult',
    method: 'POST',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getShareV1ConsultList = (
  query: { status?: ConsultStatus | ''; page?: number; per_page?: number } = {},
  params: RequestParams = {},
) =>
  httpRequest<
    DomainResponse & {
      data?: { items: ConsultInquiryListItem[]; total: number };
    }
  >({
    path: '/share/pro/v1/consult/list',
    method: 'GET',
    query,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const getShareV1ConsultDetail = (
  query: { id: number },
  params: RequestParams = {},
) =>
  httpRequest<DomainResponse & { data?: ConsultInquiryDetail }>({
    path: '/share/pro/v1/consult/detail',
    method: 'GET',
    query,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const postShareV1ConsultReply = (
  body: {
    inquiry_id: number;
    content: string;
    attachments?: ConsultAttachment[];
  },
  params: RequestParams = {},
) =>
  httpRequest<DomainResponse & { data?: { item: ConsultMessage } }>({
    path: '/share/pro/v1/consult/reply',
    method: 'POST',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });

export const postShareV1ConsultClose = (
  body: { inquiry_id: number },
  params: RequestParams = {},
) =>
  httpRequest<DomainResponse>({
    path: '/share/pro/v1/consult/close',
    method: 'POST',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
