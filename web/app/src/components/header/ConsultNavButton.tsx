'use client';

import { useBasePath } from '@/hooks';
import { useStore } from '@/provider';
import { getShareV1ConsultList } from '@/request/ShareConsult';
import { isAuthInfoEmpty } from '@/utils/authInfo';
import { lacksAccountIdentityForSiteFeedback } from '@/utils/siteFeedbackAuth';
import { Badge, Button } from '@mui/material';
import { message } from '@ctzhian/ui';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

export const CONSULT_REPLIED_COUNT_EVENT = 'consult-replied-count-changed';

export const notifyConsultRepliedCountChanged = () => {
  window.dispatchEvent(new Event(CONSULT_REPLIED_COUNT_EVENT));
};

const ConsultNavButton = () => {
  const router = useRouter();
  const basePath = useBasePath();
  const { authInfo, setLoginModalOpen } = useStore();
  const [repliedCount, setRepliedCount] = useState(0);

  const canFetch =
    !isAuthInfoEmpty(authInfo) &&
    !lacksAccountIdentityForSiteFeedback(authInfo);

  const loadRepliedCount = useCallback(async () => {
    if (!canFetch) {
      setRepliedCount(0);
      return;
    }
    try {
      const res = await getShareV1ConsultList({
        status: 'replied',
        page: 1,
        per_page: 1,
      });
      const data = res as { total?: number };
      setRepliedCount(Math.max(0, data?.total ?? 0));
    } catch {
      // 未登录或 session 失效时静默忽略
    }
  }, [canFetch]);

  useEffect(() => {
    void loadRepliedCount();
    const timer = window.setInterval(() => void loadRepliedCount(), 60000);
    return () => window.clearInterval(timer);
  }, [loadRepliedCount]);

  useEffect(() => {
    const onRefresh = () => void loadRepliedCount();
    window.addEventListener(CONSULT_REPLIED_COUNT_EVENT, onRefresh);
    return () =>
      window.removeEventListener(CONSULT_REPLIED_COUNT_EVENT, onRefresh);
  }, [loadRepliedCount]);

  const openConsult = () => {
    if (isAuthInfoEmpty(authInfo)) {
      setLoginModalOpen?.(true);
      return;
    }
    if (lacksAccountIdentityForSiteFeedback(authInfo)) {
      message.error(
        '疑难咨询需使用账号登录（用户名密码或企业 SSO），当前为访问口令认证',
      );
      setLoginModalOpen?.(true);
      return;
    }
    router.push(`${basePath}/consult`);
  };

  return (
    <Badge
      color='success'
      overlap='rectangular'
      invisible={repliedCount <= 0}
      badgeContent={repliedCount > 99 ? '99+' : repliedCount}
      sx={{
        '& .MuiBadge-badge': {
          fontSize: 11,
          height: 18,
          minWidth: 18,
          fontWeight: 600,
          top: 4,
          right: -2,
        },
      }}
    >
      <Button
        size='small'
        color='inherit'
        onClick={openConsult}
        sx={{ textTransform: 'none', whiteSpace: 'nowrap' }}
      >
        疑难咨询
      </Button>
    </Badge>
  );
};

export default ConsultNavButton;
