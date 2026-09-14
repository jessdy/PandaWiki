import Card from '@/components/Card';
import { tableSx } from '@/constant/styles';
import {
  ExpertProfileItem,
  getApiV1ExpertList,
  putApiV1ExpertProfile,
} from '@/request/ExpertProfile';
import {
  Alert,
  Box,
  Button,
  Chip,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import { Ellipsis, message, Table } from '@ctzhian/ui';
import { ColumnsType } from '@ctzhian/ui/dist/Table';
import { useCallback, useEffect, useMemo, useState } from 'react';

type ExpertDraft = Pick<ExpertProfileItem, 'name' | 'introduction'>;

const ExpertPage = () => {
  const [data, setData] = useState<ExpertProfileItem[]>([]);
  const [drafts, setDrafts] = useState<Record<string, ExpertDraft>>({});
  const [loading, setLoading] = useState(false);
  const [savingUserID, setSavingUserID] = useState('');
  const [loadError, setLoadError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError('');
    try {
      const result = await getApiV1ExpertList();
      const items = result?.items ?? [];
      setData(items);
      setDrafts(
        Object.fromEntries(
          items.map(item => [
            item.user_id,
            { name: item.name || '', introduction: item.introduction || '' },
          ]),
        ),
      );
    } catch (error: unknown) {
      const status = (
        error as { response?: { status?: number }; status?: number }
      )?.response?.status;
      setLoadError(
        status === 403
          ? '只有超级管理员可以维护专家信息。'
          : '专家信息加载失败，请稍后重试。',
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const changeDraft = (
    userID: string,
    field: keyof ExpertDraft,
    value: string,
  ) => {
    setDrafts(previous => ({
      ...previous,
      [userID]: {
        ...(previous[userID] || { name: '', introduction: '' }),
        [field]: value,
      },
    }));
  };

  const save = async (item: ExpertProfileItem) => {
    const draft = drafts[item.user_id] || { name: '', introduction: '' };
    setSavingUserID(item.user_id);
    try {
      await putApiV1ExpertProfile(item.user_id, draft);
      const next = {
        name: draft.name.trim(),
        introduction: draft.introduction.trim(),
      };
      setData(previous =>
        previous.map(row =>
          row.user_id === item.user_id ? { ...row, ...next } : row,
        ),
      );
      setDrafts(previous => ({ ...previous, [item.user_id]: next }));
      message.success('专家信息已保存');
    } catch (error: unknown) {
      message.error(
        (error as { message?: string })?.message || '专家信息保存失败',
      );
    } finally {
      setSavingUserID('');
    }
  };

  const columns = useMemo<ColumnsType<ExpertProfileItem>>(
    () => [
      {
        dataIndex: 'account',
        title: '管理员账号',
        width: 180,
        render: (value: string) => (
          <Ellipsis sx={{ maxWidth: 160 }}>{value}</Ellipsis>
        ),
      },
      {
        dataIndex: 'role',
        title: '咨询管理权限',
        width: 240,
        render: (_: string, item: ExpertProfileItem) =>
          item.role === 'admin' ? (
            <Chip size='small' color='primary' label='超级管理员' />
          ) : (
            <Tooltip
              title={
                item.consult_kb_names.length > 0
                  ? item.consult_kb_names.join('、')
                  : '咨询管理'
              }
            >
              <Box>
                <Ellipsis sx={{ maxWidth: 210 }}>
                  {item.consult_kb_names.join('、') || '咨询管理'}
                </Ellipsis>
              </Box>
            </Tooltip>
          ),
      },
      {
        dataIndex: 'name',
        title: '姓名',
        width: 220,
        render: (_: string, item: ExpertProfileItem) => (
          <TextField
            fullWidth
            size='small'
            placeholder='请输入专家姓名'
            value={drafts[item.user_id]?.name ?? ''}
            inputProps={{ maxLength: 50 }}
            onChange={event =>
              changeDraft(item.user_id, 'name', event.target.value)
            }
          />
        ),
      },
      {
        dataIndex: 'introduction',
        title: '介绍',
        render: (_: string, item: ExpertProfileItem) => {
          const value = drafts[item.user_id]?.introduction ?? '';
          return (
            <TextField
              fullWidth
              size='small'
              multiline
              minRows={2}
              maxRows={4}
              placeholder='请输入专家介绍'
              value={value}
              inputProps={{ maxLength: 500 }}
              helperText={`${Array.from(value).length}/500`}
              onChange={event =>
                changeDraft(item.user_id, 'introduction', event.target.value)
              }
            />
          );
        },
      },
      {
        dataIndex: 'operation',
        title: '操作',
        width: 100,
        render: (_: unknown, item: ExpertProfileItem) => {
          const draft = drafts[item.user_id] || {
            name: '',
            introduction: '',
          };
          const unchanged =
            draft.name === item.name &&
            draft.introduction === item.introduction;
          return (
            <Button
              size='small'
              variant='contained'
              disabled={unchanged || !!savingUserID}
              loading={savingUserID === item.user_id}
              onClick={() => void save(item)}
            >
              保存
            </Button>
          );
        },
      },
    ],
    [drafts, savingUserID],
  );

  return (
    <Card>
      <Stack sx={{ p: 2 }} spacing={0.5}>
        <Typography variant='h6' fontWeight={600}>
          专家信息
        </Typography>
        <Typography variant='body2' color='text.secondary'>
          维护具备咨询管理权限的管理员对外展示姓名和介绍。权限变更后，名单会自动同步。
        </Typography>
      </Stack>

      {loadError && (
        <Alert
          severity='error'
          action={
            <Button color='inherit' size='small' onClick={() => void load()}>
              重试
            </Button>
          }
          sx={{ mx: 2, mb: 2 }}
        >
          {loadError}
        </Alert>
      )}

      <Box sx={{ px: 2, pb: 2 }}>
        <Table
          rowKey='user_id'
          columns={columns}
          dataSource={data}
          size='small'
          loading={loading}
          pagination={false}
          sx={{
            ...tableSx,
            '.MuiTableCell-root': { verticalAlign: 'top' },
          }}
          renderEmpty={
            loading ? (
              <Box />
            ) : (
              <Stack alignItems='center' sx={{ py: 8 }}>
                <Typography color='text.secondary'>
                  暂无具有咨询管理权限的管理员
                </Typography>
              </Stack>
            )
          }
        />
      </Box>
    </Card>
  );
};

export default ExpertPage;
