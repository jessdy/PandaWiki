import { postApiV1UserLogin } from '@/request/User';
import Bgi from '@/assets/images/login-bgi.png';
import Avatar from '@/components/Avatar';
import Card from '@/components/Card';
import { Box, Button, IconButton, Stack, TextField } from '@mui/material';
import { message } from '@ctzhian/ui';
import { useMemo, useState } from 'react';
import { useAdminSiteBranding } from '@/hooks/useAdminSiteBranding';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  IconZhanghao,
  IconIcon_tool_close,
  IconMima,
  IconKejian,
  IconBukejian,
} from '@panda-wiki/icons';
import ClassificationBanner from '@/components/ClassificationBanner';
import { markAuthed, setLoginPortal, type LoginPortal } from '@/utils/auth';
import { ConstsUserRole } from '@/request/types';

const PORTAL_TITLE: Record<LoginPortal, string> = {
  admin: '',
  security: '安全管理员登录',
  audit: '审计管理员登录',
};

const portalFromPath = (pathname: string): LoginPortal => {
  if (pathname.includes('/login/security')) return 'security';
  if (pathname.includes('/login/audit')) return 'audit';
  return 'admin';
};

const homeForRole = (role?: string) => {
  if (role === ConstsUserRole.UserRoleSecurityAdmin) return '/approval';
  if (role === ConstsUserRole.UserRoleAuditAdmin) return '/audit';
  return '/';
};

const Login = () => {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const portal = portalFromPath(pathname);
  const kbId = useMemo(
    () =>
      typeof window !== 'undefined' ? localStorage.getItem('kb_id') : null,
    [],
  );
  const { displayTitle, logoSrc } = useAdminSiteBranding(kbId);
  const [account, setAccount] = useState('');
  const [password, setPassword] = useState('');
  const [see, setSee] = useState(false);
  const [loading, setLoading] = useState(false);

  const title = PORTAL_TITLE[portal] || displayTitle;

  const submit = () => {
    postApiV1UserLogin({ account, password, portal })
      .then(res => {
        // JWT 由服务端写入 HttpOnly Cookie，前端只保留登录态标记
        setLoginPortal(portal);
        markAuthed();
        navigate(homeForRole(res?.role));
        message.success('登录成功');
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <Box
      sx={{
        width: '100vw',
        height: '100vh',
        backgroundImage: `url(${Bgi})`,
        backgroundSize: 'cover',
        backgroundPosition: 'center',
        backgroundRepeat: 'no-repeat',
      }}
    >
      <ClassificationBanner />
      <Stack
        direction='row'
        alignItems={'center'}
        justifyContent={'center'}
        sx={{
          width: '100%',
          height: '100%',
        }}
      >
        <Card
          sx={{
            mt: -3,
            p: 4,
            width: 458,
            boxShadow:
              '0px 0px 4px 0px rgba(54,59,76,0.1), 0px 20px 40px 0px rgba(54,59,76,0.1)',
            position: 'absolute',
            top: '50%',
            left: '50%',
            bgcolor: 'background.paper',
            transform: 'translate(-50%, -50%)',
          }}
        >
          <Stack alignItems={'center'}>
            <Avatar src={logoSrc} sx={{ width: 64, height: 64, mb: 1 }} />
            <Stack
              direction={'row'}
              alignItems={'center'}
              gap={1}
              sx={{
                fontSize: 28,
                fontWeight: 700,
                color: 'text.primary',
                mb: 4,
              }}
            >
              {title || displayTitle}
            </Stack>
            <TextField
              value={account}
              fullWidth
              sx={{ mb: 4 }}
              onChange={e => setAccount(e.target.value)}
              placeholder='账号'
              autoFocus
              tabIndex={1}
              slotProps={{
                input: {
                  startAdornment: (
                    <IconZhanghao sx={{ fontSize: 16, mr: 2, flexShrink: 0 }} />
                  ),
                  endAdornment: account ? (
                    <IconButton
                      onClick={() => setAccount('')}
                      size='small'
                      tabIndex={-1}
                    >
                      <IconIcon_tool_close
                        sx={{ fontSize: 14, color: 'text.tertiary' }}
                      />
                    </IconButton>
                  ) : null,
                },
              }}
            />
            <TextField
              value={password}
              fullWidth
              sx={{ mb: 4 }}
              onChange={e => setPassword(e.target.value)}
              tabIndex={2}
              onKeyDown={e => {
                if (e.key === 'Enter') {
                  if (!account || !password) return;
                  setLoading(true);
                  submit();
                }
              }}
              placeholder='密码'
              type={see ? 'text' : 'password'}
              slotProps={{
                input: {
                  startAdornment: (
                    <IconMima sx={{ fontSize: 16, mr: 2, flexShrink: 0 }} />
                  ),
                  endAdornment: password ? (
                    <Stack direction={'row'} alignItems={'center'}>
                      <IconButton
                        onClick={() => setSee(!see)}
                        size='small'
                        tabIndex={-1}
                      >
                        {see ? (
                          <IconKejian
                            sx={{ fontSize: 18, color: 'text.tertiary' }}
                          />
                        ) : (
                          <IconBukejian
                            sx={{ fontSize: 18, color: 'text.tertiary' }}
                          />
                        )}
                      </IconButton>
                      <IconButton
                        onClick={() => setPassword('')}
                        size='small'
                        tabIndex={-1}
                      >
                        <IconIcon_tool_close
                          sx={{ fontSize: 14, color: 'text.tertiary' }}
                        />
                      </IconButton>
                    </Stack>
                  ) : null,
                },
              }}
            />
            <Button
              fullWidth
              variant='contained'
              sx={{ height: 48 }}
              onClick={() => {
                if (!account || !password) return;
                setLoading(true);
                submit();
              }}
              loading={loading}
            >
              登录
            </Button>
          </Stack>
        </Card>
      </Stack>
    </Box>
  );
};

export default Login;
