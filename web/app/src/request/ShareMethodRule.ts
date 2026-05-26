import httpRequest, {
  ContentType,
  DomainResponse,
  RequestParams,
} from './httpClient';

/** 命中的开封方法（含文档名，用于卡片渲染）。 */
export interface MethodMatchView {
  id: string;
  name: string;
  description?: string;
  node_id: string;
  node_name?: string;
}

/** 品类的属性 + 枚举值定义；前台属性面板按此渲染 Select。 */
export interface MatchAttrSpec {
  name: string;
  values?: string[];
}

export interface MatchResp {
  category: string;
  specs: MatchAttrSpec[];
  methods: MethodMatchView[];
}

/**
 * 前台属性面板的实时联动接口。
 * 不走 LLM、不写 conversation，由用户每次调整 Select 后调用。
 */
export const postShareV1MethodRulesMatch = (
  body: { category: string; collected: Record<string, string> },
  params: RequestParams = {},
) =>
  httpRequest<DomainResponse & { data?: MatchResp }>({
    path: '/share/v1/method_rules/match',
    method: 'POST',
    body,
    type: ContentType.Json,
    format: 'json',
    ...params,
  });
