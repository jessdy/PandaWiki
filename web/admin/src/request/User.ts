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
  DeleteApiV1UserDeleteParams,
  DomainPWResponse,
  DomainResponse,
  V1CreateUserReq,
  V1CreateUserResp,
  V1LoginReq,
  V1LoginResp,
  V1ResetPasswordReq,
  V1UserInfoResp,
  V1UserListResp,
} from "./types";

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

/**
 * @description CreateUser
 *
 * @tags user
 * @name PostApiV1UserCreate
 * @summary CreateUser
 * @request POST:/api/v1/user/create
 * @response `200` `(DomainResponse & {
    data?: V1CreateUserResp,

})` OK
 */

export const postApiV1UserCreate = (
  body: V1CreateUserReq,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainResponse & {
      data?: V1CreateUserResp;
    }
  >({
    path: `/api/v1/user/create`,
    method: "POST",
    body: body,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description DeleteUser
 *
 * @tags user
 * @name DeleteApiV1UserDelete
 * @summary DeleteUser
 * @request DELETE:/api/v1/user/delete
 * @response `200` `DomainResponse` OK
 */

export const deleteApiV1UserDelete = (
  query: DeleteApiV1UserDeleteParams,
  params: RequestParams = {},
) =>
  httpRequest<DomainResponse>({
    path: `/api/v1/user/delete`,
    method: "DELETE",
    query: query,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description ListUsers
 *
 * @tags user
 * @name GetApiV1UserList
 * @summary ListUsers
 * @request GET:/api/v1/user/list
 * @response `200` `(DomainPWResponse & {
    data?: V1UserListResp,

})` OK
 */

export const getApiV1UserList = (params: RequestParams = {}) =>
  httpRequest<
    DomainPWResponse & {
      data?: V1UserListResp;
    }
  >({
    path: `/api/v1/user/list`,
    method: "GET",
    type: ContentType.Json,
    format: "json",
    ...params,
  });

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
 * @description ResetPassword
 *
 * @tags user
 * @name PutApiV1UserResetPassword
 * @summary ResetPassword
 * @request PUT:/api/v1/user/reset_password
 * @response `200` `DomainResponse` OK
 */

export const putApiV1UserResetPassword = (
  body: V1ResetPasswordReq,
  params: RequestParams = {},
) =>
  httpRequest<DomainResponse>({
    path: `/api/v1/user/reset_password`,
    method: "PUT",
    body: body,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description ListGuestUsers
 *
 * @tags user
 * @name GetApiV1UserGuestList
 * @summary ListGuestUsers
 * @request GET:/api/v1/user/guest/list
 * @response `200` `(DomainPWResponse & {
    data?: V1UserListResp,

})` OK
 */

export const getApiV1UserGuestList = (params: RequestParams = {}) =>
  httpRequest<
    DomainPWResponse & {
      data?: V1UserListResp;
    }
  >({
    path: `/api/v1/user/guest/list`,
    method: "GET",
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description CreateGuestUser
 *
 * @tags user
 * @name PostApiV1UserGuestCreate
 * @summary CreateGuestUser
 * @request POST:/api/v1/user/guest/create
 * @response `200` `(DomainResponse & {
    data?: V1CreateUserResp,

})` OK
 */

export const postApiV1UserGuestCreate = (
  body: V1CreateUserReq,
  params: RequestParams = {},
) =>
  httpRequest<
    DomainResponse & {
      data?: V1CreateUserResp;
    }
  >({
    path: `/api/v1/user/guest/create`,
    method: "POST",
    body: body,
    type: ContentType.Json,
    format: "json",
    ...params,
  });

/**
 * @description UpdateGuestUser
 *
 * @tags user
 * @name PutApiV1UserGuestId
 * @summary UpdateGuestUser
 * @request PUT:/api/v1/user/guest/:id
 * @response `200` `DomainResponse` OK
 */

export const putApiV1UserGuestId = (
  params: {
    id: string;
    body: V1CreateUserReq;
  },
  requestParams: RequestParams = {},
) =>
  httpRequest<DomainResponse>({
    path: `/api/v1/user/guest/${params.id}`,
    method: "PUT",
    body: params.body,
    type: ContentType.Json,
    format: "json",
    ...requestParams,
  });

/**
 * @description DeleteGuestUser
 *
 * @tags user
 * @name DeleteApiV1UserGuestId
 * @summary DeleteGuestUser
 * @request DELETE:/api/v1/user/guest/:id
 * @response `200` `DomainResponse` OK
 */

export const deleteApiV1UserGuestId = (
  params: {
    id: string;
  },
  requestParams: RequestParams = {},
) =>
  httpRequest<DomainResponse>({
    path: `/api/v1/user/guest/${params.id}`,
    method: "DELETE",
    type: ContentType.Json,
    format: "json",
    ...requestParams,
  });

// 用户组管理 API
export const getApiV1UserAuthGroupList = (params: { kb_id?: string } = {}) =>
  httpRequest<any>({
    path: `/api/v1/user/auth_group/list`,
    method: "GET",
    type: ContentType.Json,
    format: "json",
    query: params,
  });

export const postApiV1UserAuthGroupCreate = (body: any) =>
  httpRequest<any>({
    path: `/api/v1/user/auth_group/create`,
    method: "POST",
    body: body,
    type: ContentType.Json,
    format: "json",
  });

export const putApiV1UserAuthGroupId = (id: number, body: any) =>
  httpRequest<any>({
    path: `/api/v1/user/auth_group/${id}`,
    method: "PUT",
    body: body,
    type: ContentType.Json,
    format: "json",
  });

export const deleteApiV1UserAuthGroupId = (id: number) =>
  httpRequest<any>({
    path: `/api/v1/user/auth_group/${id}`,
    method: "DELETE",
    type: ContentType.Json,
    format: "json",
  });

// 获取用户所属的用户组
export const getApiV1UserGroups = (params: { user_id: string }) =>
  httpRequest<any>({
    path: `/api/v1/user/groups`,
    method: "GET",
    type: ContentType.Json,
    format: "json",
    query: params,
  });

// 更新用户所属的用户组
export const putApiV1UserGroups = (body: { user_id: string; group_ids: number[] }) =>
  httpRequest<any>({
    path: `/api/v1/user/groups`,
    method: "PUT",
    body: body,
    type: ContentType.Json,
    format: "json",
  });
