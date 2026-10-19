import {
  CloudAuthSetting,
  getApiV1SystemCloudAuth,
  putApiV1SystemCloudAuth,
} from '@/request/CloudAuth';
import { message, Modal } from '@ctzhian/ui';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { forwardRef, useEffect, useImperativeHandle, useState } from 'react';

export interface CloudAuthConfigRef {
  hasUnsavedChanges: () => boolean;
  handleClose: () => void;
}

interface CloudAuthConfigProps {
  onCloseModal: () => void;
}

const emptySetting = (): CloudAuthSetting => ({ host: '', port: 0 });

const CloudAuthConfig = forwardRef<CloudAuthConfigRef, CloudAuthConfigProps>(
  ({ onCloseModal }, ref) => {
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [loadError, setLoadError] = useState(false);
    const [host, setHost] = useState('');
    const [port, setPort] = useState('');
    const [saved, setSaved] = useState<CloudAuthSetting>(emptySetting());

    const applySetting = (setting: CloudAuthSetting) => {
      const next = {
        host: setting.host || '',
        port: setting.port || 0,
      };
      setSaved(next);
      setHost(next.host);
      setPort(next.port ? String(next.port) : '');
    };

    const load = () => {
      setLoading(true);
      setLoadError(false);
      getApiV1SystemCloudAuth()
        .then(res => applySetting(res || emptySetting()))
        .catch(() => setLoadError(true))
        .finally(() => setLoading(false));
    };

    useEffect(() => {
      load();
    }, []);

    const currentPort = port.trim() === '' ? 0 : Number(port);
    const dirty = host.trim() !== saved.host || currentPort !== saved.port;

    const handleClose = () => {
      if (!dirty) {
        onCloseModal();
        return;
      }
      Modal.confirm({
        title: '有未保存的平台地址',
        content: '关闭后本次修改将丢失，是否继续？',
        okText: '确认关闭',
        cancelText: '继续编辑',
        onOk: onCloseModal,
      });
    };

    useImperativeHandle(ref, () => ({
      hasUnsavedChanges: () => dirty,
      handleClose,
    }));

    const handleSave = async () => {
      if (port.trim() !== '' && !Number.isInteger(currentPort)) {
        message.error('端口必须是整数');
        return;
      }
      setSaving(true);
      try {
        const result = await putApiV1SystemCloudAuth({
          host: host.trim(),
          port: currentPort,
        });
        applySetting(result || emptySetting());
        message.success('平台地址已保存');
      } catch {
        // 请求拦截器已经提示错误。
      } finally {
        setSaving(false);
      }
    };

    if (loading) {
      return (
        <Stack
          alignItems='center'
          justifyContent='center'
          sx={{ minHeight: 360 }}
        >
          <CircularProgress size={28} />
        </Stack>
      );
    }

    if (loadError) {
      return (
        <Stack
          alignItems='center'
          justifyContent='center'
          gap={2}
          sx={{ minHeight: 360 }}
        >
          <Alert severity='error'>平台地址加载失败</Alert>
          <Button variant='outlined' onClick={load}>
            重新加载
          </Button>
        </Stack>
      );
    }

    return (
      <Stack gap={2.5}>
        <Box>
          <Typography variant='h6' sx={{ fontWeight: 600 }}>
            平台地址
          </Typography>
          <Typography variant='body2' color='text.secondary' sx={{ mt: 0.5 }}>
            用于 `/auth/login?tgt=` 的会话验证和权限验证。系统会请求
            http://IP:端口 下的 loginVerify 和 checkPermissions。
          </Typography>
        </Box>
        <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
          <TextField
            label='平台 IP'
            value={host}
            onChange={event => setHost(event.target.value)}
            placeholder='10.10.10.252'
            fullWidth
          />
          <TextField
            label='端口'
            value={port}
            onChange={event => setPort(event.target.value.replace(/[^\d]/g, ''))}
            placeholder='8080'
            sx={{ width: { sm: 180 } }}
          />
        </Stack>
        <Stack direction='row' justifyContent='flex-end'>
          <Button
            variant='contained'
            loading={saving}
            disabled={!dirty}
            onClick={handleSave}
          >
            保存设置
          </Button>
        </Stack>
      </Stack>
    );
  },
);

CloudAuthConfig.displayName = 'CloudAuthConfig';

export default CloudAuthConfig;
