import {
  getApiV1SystemTestEnv,
  putApiV1SystemTestEnv,
} from '@/request/TestEnv';
import { message } from '@ctzhian/ui';
import {
  Alert,
  Box,
  CircularProgress,
  FormControlLabel,
  Stack,
  Switch,
  Typography,
} from '@mui/material';
import { useEffect, useState } from 'react';

interface TestEnvConfigProps {
  onEnabledChange?: (enabled: boolean) => void;
}

const TestEnvConfig = ({ onEnabledChange }: TestEnvConfigProps) => {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const [enabled, setEnabled] = useState(false);

  const load = () => {
    setLoading(true);
    setLoadError(false);
    getApiV1SystemTestEnv()
      .then(res => {
        const next = !!res?.enabled;
        setEnabled(next);
        onEnabledChange?.(next);
      })
      .catch(() => setLoadError(true))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
    // 只在打开配置时读取一次。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleToggle = async (next: boolean) => {
    const previous = enabled;
    setEnabled(next);
    setSaving(true);
    try {
      const saved = await putApiV1SystemTestEnv({ enabled: next });
      const value = !!saved?.enabled;
      setEnabled(value);
      onEnabledChange?.(value);
      message.success(value ? '测试环境已打开' : '测试环境已关闭');
    } catch {
      setEnabled(previous);
      message.error('测试环境开关保存失败');
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
        <Alert severity='error'>测试环境配置加载失败</Alert>
      </Stack>
    );
  }

  return (
    <Stack gap={2.5}>
      <Box>
        <Typography variant='h6' sx={{ fontWeight: 600 }}>
          测试环境
        </Typography>
        <Typography variant='body2' color='text.secondary' sx={{ mt: 0.5 }}>
          打开后，除实战模式外的问答不再调用向量模型和重排模型。系统会从当前知识库允许被问答的文档中随机抽取
          TopN 篇（沿用本次提问的检索条数，未指定时为 10
          篇），未发布文档也使用当前正文，交给对话模型阅读后回答。对话模型仍然需要配置。实战模式始终按向量检索和重排回答，不受此开关影响。
        </Typography>
      </Box>
      <FormControlLabel
        control={
          <Switch
            checked={enabled}
            disabled={saving}
            onChange={(_, checked) => handleToggle(checked)}
          />
        }
        label={enabled ? '测试环境已打开' : '测试环境已关闭'}
      />
      {enabled && (
        <Alert severity='warning'>
          除实战模式外，当前回答不按问题相关性检索，只随机抽取文档。实战模式仍使用向量检索和重排。关闭开关后，其他模式也会恢复向量检索和重排。
        </Alert>
      )}
    </Stack>
  );
};

export default TestEnvConfig;
