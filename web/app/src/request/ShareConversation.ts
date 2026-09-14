/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

import httpRequest, { ContentType, RequestParams } from "./httpClient";
import {
  DomainPWResponse,
  DomainShareConversationDetailResp,
  GetShareV1ConversationDetailParams,
} from "./types";

/**
 * @description GetConversationDetail
 *
 * @tags share_conversation
 * @name GetShareV1ConversationDetail
 * @summary GetConversationDetail
 * @request GET:/share/v1/conversation/detail
 * @response `200` `(DomainPWResponse & {
    data?: DomainShareConversationDetailResp,

})` OK
 */

export const getShareV1ConversationDetail = (
  query: GetShareV1ConversationDetailParams,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: DomainShareConversationDetailResp;
    }
  >({
    path: `/share/v1/conversation/detail`,
    method: "GET",
    query: query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

export interface ShareConversationHistoryItem {
  id: string;
  subject: string;
  created_at: string;
}

export interface ShareConversationHistoryResp {
  available: boolean;
  total: number;
  data: ShareConversationHistoryItem[];
}

export const getShareV1ConversationHistory = (
  query: { page: number; per_page: number },
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: ShareConversationHistoryResp;
    }
  >({
    path: `/share/v1/conversation/history`,
    method: "GET",
    query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

export const getShareV1ConversationHistoryDetail = (
  query: { id: string },
  params: RequestParams = {},
) =>
  httpRequest<
    DomainPWResponse & {
      data?: DomainShareConversationDetailResp;
    }
  >({
    path: `/share/v1/conversation/history/detail`,
    method: "GET",
    query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });
