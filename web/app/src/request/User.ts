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
  V1LoginReq,
  V1LoginResp,
  V1UserInfoResp,
} from "./types";

/**
 * @description Login
 *
 * @tags user
 * @name PostApiV1UserLogin
 * @summary Login
 * @request POST:/api/v1/user/login
 * @response `200` `V1LoginResp` OK
 */
export const postApiV1UserLogin = (
  body: V1LoginReq,
  params: RequestParams = {},
) =>
  httpRequest<V1LoginResp>({
    path: `/api/v1/user/login`,
    method: "POST",
    body: body,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description GetUser
 *
 * @tags user
 * @name GetApiV1User
 * @summary GetUser
 * @request GET:/api/v1/user
 * @response `200` `V1UserInfoResp` OK
 */
export const getApiV1User = (params: RequestParams = {}) =>
  httpRequest<V1UserInfoResp>({
    path: `/api/v1/user`,
    method: "GET",
    type: ContentType.Json,
    ...params,
  });

