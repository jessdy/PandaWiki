import { Box } from '@mui/material';
import { Outlet, useLocation } from 'react-router-dom';
import Header from '@/components/Header';
import Sidebar from '@/components/Sidebar';
import ClassificationBanner from '@/components/ClassificationBanner';
import CreateWikiModal from '@/components/CreateWikiModal';
import { getApiV1ModelList } from '@/request/Model';
import { getApiV1SystemTestEnv } from '@/request/TestEnv';
import { getApiV1KnowledgeBaseList } from '@/request/KnowledgeBase';
import { getApiV1User } from '@/request/User';
import { useAppDispatch } from '@/store';
import {
  setModelStatus,
  setModelList,
  setKbList,
  setKbId,
  setUser,
} from '@/store/slices/config';
import { ConstsUserRole } from '@/request/types';
import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

const isSeparatedAdmin = (role?: string) =>
  role === ConstsUserRole.UserRoleSecurityAdmin ||
  role === ConstsUserRole.UserRoleAuditAdmin;

const useAuth = (hasAuth: boolean) => {
  const { pathname } = useLocation();
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const getModel = () => {
    return Promise.all([
      getApiV1ModelList(),
      getApiV1SystemTestEnv().catch(() => ({ enabled: false })),
    ]).then(([res, testEnv]) => {
      // @ts-expect-error 类型不匹配
      const chat = res.find(it => it.type === 'chat') || null;
      // @ts-expect-error 类型不匹配
      const embedding = res.find(it => it.type === 'embedding') || null;
      // @ts-expect-error 类型不匹配
      const rerank = res.find(it => it.type === 'rerank') || null;
      const status = testEnv?.enabled
        ? !!chat
        : !!(chat && embedding && rerank);
      dispatch(setModelStatus(status));
      dispatch(setModelList(res));
      return status;
    });
  };

  const getKbList = (id?: string) => {
    const kb_id = id || localStorage.getItem('kb_id') || '';
    return getApiV1KnowledgeBaseList().then(res => {
      dispatch(setKbList(res));
      if (res.find(item => item.id === kb_id)) {
        dispatch(setKbId(kb_id));
      } else {
        dispatch(setKbId(res[0]?.id || ''));
      }
      return res;
    });
  };

  const getUser = () => {
    return getApiV1User().then(res => {
      dispatch(setUser(res));
      return res;
    });
  };

  const initData = () => {
    getUser().then(user => {
      if (isSeparatedAdmin(user.role)) {
        if (
          user.role === ConstsUserRole.UserRoleSecurityAdmin &&
          pathname === '/'
        ) {
          navigate('/approval');
        }
        if (
          user.role === ConstsUserRole.UserRoleAuditAdmin &&
          pathname === '/'
        ) {
          navigate('/audit');
        }
        return;
      }
      Promise.all([
        user.role === ConstsUserRole.UserRoleAdmin
          ? getModel()
          : Promise.resolve(null),
        getKbList(),
      ]).then(([, kbList]) => {
        if (
          user.role !== ConstsUserRole.UserRoleAdmin &&
          user.role !== ConstsUserRole.UserRoleUser &&
          kbList.length === 0 &&
          pathname !== '/login'
        ) {
          navigate('401');
        }
      });
    });
  };

  useEffect(() => {
    if (hasAuth) {
      initData();
    }
  }, [hasAuth]);
};

export const MainLayout = () => {
  useAuth(true);
  return (
    <>
      <ClassificationBanner />
      <Box
        sx={{
          position: 'relative',
          minWidth: '900px',
          minHeight: '100vh',
          fontSize: '16px',
          bgcolor: 'background.paper2',
        }}
      >
        <Sidebar />
        <Header />
        <Box
          sx={{
            pr: 2,
            width: 'calc(100% - 170px)',
            pt: '64px',
            ml: '170px',
            color: 'text.primary',
          }}
        >
          <Outlet />
        </Box>
      </Box>
      <CreateWikiModal />
    </>
  );
};

export const NoSidebarHeaderLayout = ({
  hasAuth = false,
}: {
  hasAuth: boolean;
}) => {
  useAuth(hasAuth);
  return (
    <Box
      sx={{
        minWidth: '900px',
      }}
    >
      <ClassificationBanner />
      <Outlet />
    </Box>
  );
};
