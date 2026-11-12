import {
  ClassificationSetting,
  getApiV1SystemClassification,
  putApiV1SystemClassification,
} from '@/request/SystemInfo';
import {
  Box,
  Button,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import { useEffect, useState } from 'react';

const ClassificationConfig = () => {
  const [setting, setSetting] = useState<ClassificationSetting>({
    enabled: false,
    level: '',
  });
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    getApiV1SystemClassification()
      .then(res => setSetting(res || { enabled: false, level: '' }))
      .catch(() => undefined);
  }, []);

  return (
    <Box>
      <Typography variant='body2' color='text.secondary' sx={{ mb: 2 }}>
        默认不显示密级标识。开启后，登录页与后台首页顶部会展示配置的密级文字。
      </Typography>
      <Stack direction='row' alignItems='center' gap={2} mb={2}>
        <Typography>显示密级</Typography>
        <Switch
          checked={setting.enabled}
          onChange={e => setSetting({ ...setting, enabled: e.target.checked })}
        />
      </Stack>
      <TextField
        fullWidth
        label='密级文字'
        placeholder='例如：内部 / 秘密 / 机密'
        value={setting.level}
        onChange={e => setSetting({ ...setting, level: e.target.value })}
        sx={{ mb: 2 }}
      />
      <Button
        variant='contained'
        disabled={loading}
        onClick={async () => {
          setLoading(true);
          try {
            const saved = await putApiV1SystemClassification(setting);
            setSetting(saved);
            message.success('密级标识已保存');
          } finally {
            setLoading(false);
          }
        }}
      >
        保存
      </Button>
    </Box>
  );
};

export default ClassificationConfig;
