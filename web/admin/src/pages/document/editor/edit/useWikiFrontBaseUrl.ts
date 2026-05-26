import type { KnowledgeBaseListItem } from '@/api';
import { getApiV1KnowledgeBaseDetail } from '@/request/KnowledgeBase';
import { useAppSelector } from '@/store';
import type { DomainAccessSettings } from '@/request/types';
import { useEffect, useState } from 'react';

function resolveFromAccessSettings(
  access?: DomainAccessSettings | null,
): string {
  if (!access) return '';
  const baseUrl = access.base_url?.trim();
  if (baseUrl) return baseUrl.replace(/\/$/, '');
  const host = access.hosts?.[0] || '';
  if (!host) return '';
  const { ssl_ports = [], ports = [] } = access;
  if (ssl_ports?.length) {
    if (ssl_ports.includes(443)) return `https://${host}`;
    return `https://${host}:${ssl_ports[0]}`;
  }
  if (ports?.length) {
    if (ports.includes(80)) return `http://${host}`;
    return `http://${host}:${ports[0]}`;
  }
  return '';
}

/**
 * 与 Header「前台查看」一致：用于生成站内文档链接。
 *
 * 解析顺序（优先级从高到低）：
 *   1. 传入的 `kbList`（管理后台缓存的知识库列表）
 *   2. Redux 中的 `kbDetail`（当前选中的知识库详情，可能比 list 更新）
 *   3. 直接调用 `GET /api/v1/knowledge_base/detail` 兜底拉取
 *
 * 任何一个解析到有效的 base_url / host 即返回；全部失败时返回空串，
 * 由调用方决定是否提示用户「未配置前台域名」。
 */
export function useWikiFrontBaseUrl(
  kbList: KnowledgeBaseListItem[] | null | undefined,
  kbId: string | undefined,
): string {
  const [wikiUrl, setWikiUrl] = useState('');
  const kbDetail = useAppSelector(state => state.config.kbDetail);

  useEffect(() => {
    if (!kbId) {
      setWikiUrl('');
      return;
    }

    let cancelled = false;

    // KnowledgeBaseAccessSettings (来自 @/api) 的字段是 nullable，而 DomainAccessSettings
    // （来自 @/request/types）是 optional。这里只读字段，做一次结构上的兼容转换。
    const fromList = kbList?.find(item => item.id === kbId)
      ?.access_settings as unknown as DomainAccessSettings | undefined;
    const listUrl = resolveFromAccessSettings(fromList);
    if (listUrl) {
      setWikiUrl(listUrl);
      return () => {
        cancelled = true;
      };
    }

    const detailUrl =
      kbDetail?.id === kbId
        ? resolveFromAccessSettings(kbDetail.access_settings)
        : '';
    if (detailUrl) {
      setWikiUrl(detailUrl);
      return () => {
        cancelled = true;
      };
    }

    setWikiUrl('');
    getApiV1KnowledgeBaseDetail({ id: kbId })
      .then(res => {
        if (cancelled) return;
        const url = resolveFromAccessSettings(res?.access_settings);
        if (url) setWikiUrl(url);
      })
      .catch(() => {});

    return () => {
      cancelled = true;
    };
  }, [kbList, kbId, kbDetail]);

  return wikiUrl;
}
