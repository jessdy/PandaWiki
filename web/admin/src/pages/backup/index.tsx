import {
  BackupInfo,
  BackupSetting,
  deleteApiV1Backup,
  getApiV1BackupList,
  getApiV1BackupSetting,
  getApiV1BackupStatus,
  getApiV1BackupStorage,
  postApiV1BackupCreate,
  postApiV1BackupRestore,
  putApiV1BackupSetting,
} from '@/request/Backup';
import { getLegacyBearerToken } from '@/utils/auth';
import {
  Box,
  Button,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { message, Modal, Table } from '@ctzhian/ui';
import { useEffect, useState } from 'react';

const formatBytes = (n: number) => {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
};

const BackupPage = () => {
  const [list, setList] = useState<BackupInfo[]>([]);
  const [setting, setSetting] = useState<BackupSetting | null>(null);
  const [storageText, setStorageText] = useState('');
  const [loading, setLoading] = useState(false);
  const [taskText, setTaskText] = useState('');

  const refresh = async () => {
    setLoading(true);
    try {
      const [listRes, settingRes, storageRes, statusRes] = await Promise.all([
        getApiV1BackupList(),
        getApiV1BackupSetting(),
        getApiV1BackupStorage(),
        getApiV1BackupStatus(),
      ]);
      setList(listRes?.list || []);
      setSetting(settingRes);
      setStorageText(
        `磁盘已用 ${storageRes.used_percent.toFixed(1)}%（备份占用 ${formatBytes(storageRes.backup_bytes)}）` +
          (storageRes.alert ? ' 【空间告警】' : '') +
          (storageRes.remind ? ' 【请尽快备份】' : ''),
      );
      setTaskText(
        statusRes.running
          ? `任务进行中：${statusRes.kind} ${statusRes.step}`
          : statusRes.error
            ? `上次失败：${statusRes.error}`
            : statusRes.name
              ? `上次完成：${statusRes.name}`
              : '',
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), 15000);
    return () => window.clearInterval(timer);
  }, []);

  return (
    <Box sx={{ p: 2 }}>
      <Stack direction='row' justifyContent='space-between' mb={2}>
        <Typography variant='h6'>数据备份与恢复</Typography>
        <Typography variant='body2' color='text.secondary'>
          {storageText} {taskText}
        </Typography>
      </Stack>
      {setting && (
        <Stack
          direction='row'
          gap={2}
          alignItems='center'
          mb={2}
          flexWrap='wrap'
        >
          <Stack direction='row' alignItems='center' gap={1}>
            <Typography>定时备份</Typography>
            <Switch
              checked={setting.enabled}
              onChange={e =>
                setSetting({ ...setting, enabled: e.target.checked })
              }
            />
          </Stack>
          <TextField
            size='small'
            label='每天时间'
            value={setting.time}
            onChange={e => setSetting({ ...setting, time: e.target.value })}
            sx={{ width: 120 }}
          />
          <TextField
            size='small'
            type='number'
            label='保留份数'
            value={setting.retention}
            onChange={e =>
              setSetting({ ...setting, retention: Number(e.target.value) || 7 })
            }
            sx={{ width: 120 }}
          />
          <Stack direction='row' alignItems='center' gap={1}>
            <Typography>包含文件</Typography>
            <Switch
              checked={setting.include_files}
              onChange={e =>
                setSetting({ ...setting, include_files: e.target.checked })
              }
            />
          </Stack>
          <Button
            variant='outlined'
            onClick={async () => {
              await putApiV1BackupSetting(setting);
              message.success('备份策略已保存');
            }}
          >
            保存策略
          </Button>
          <Button
            variant='contained'
            onClick={async () => {
              await postApiV1BackupCreate();
              message.success('备份已开始');
              void refresh();
            }}
          >
            立即备份
          </Button>
        </Stack>
      )}
      <Table
        loading={loading}
        dataSource={list}
        rowKey='name'
        columns={[
          { title: '名称', dataIndex: 'name' },
          {
            title: '大小',
            dataIndex: 'size',
            width: 100,
            render: (v: number) => formatBytes(v),
          },
          { title: '时间', dataIndex: 'created_at', width: 180 },
          { title: '触发', dataIndex: 'trigger', width: 100 },
          {
            title: '表/行/文件',
            dataIndex: 'tables',
            width: 140,
            render: (_: unknown, r: BackupInfo) =>
              `${r.tables}/${r.rows}/${r.files}`,
          },
          {
            title: '操作',
            dataIndex: 'name',
            width: 240,
            render: (_: unknown, row: BackupInfo) => (
              <Stack direction='row' gap={1}>
                <Button
                  size='small'
                  onClick={async () => {
                    const token = getLegacyBearerToken();
                    const headers: Record<string, string> = {};
                    if (token) {
                      headers.Authorization = `Bearer ${token}`;
                    }
                    const resp = await fetch(
                      `${window.__BASENAME__ || ''}/api/v1/backup/${encodeURIComponent(row.name)}/download`,
                      { credentials: 'include', headers },
                    );
                    if (!resp.ok) {
                      message.error('下载失败');
                      return;
                    }
                    const blob = await resp.blob();
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = row.name;
                    a.click();
                    URL.revokeObjectURL(url);
                  }}
                >
                  下载
                </Button>
                <Button
                  size='small'
                  color='warning'
                  onClick={() => {
                    Modal.confirm({
                      title: '确认恢复？',
                      content: `将先自动备份当前数据，再恢复 ${row.name}。请输入 RESTORE 确认。`,
                      onOk: async () => {
                        const confirm =
                          window.prompt('请输入 RESTORE 确认恢复') || '';
                        await postApiV1BackupRestore({
                          name: row.name,
                          confirm,
                        });
                        message.success('恢复完成');
                        void refresh();
                      },
                    });
                  }}
                >
                  恢复
                </Button>
                <Button
                  size='small'
                  color='error'
                  onClick={() => {
                    Modal.confirm({
                      title: '删除备份',
                      content: `确认删除 ${row.name}？`,
                      onOk: async () => {
                        await deleteApiV1Backup(row.name);
                        message.success('已删除');
                        void refresh();
                      },
                    });
                  }}
                >
                  删除
                </Button>
              </Stack>
            ),
          },
        ]}
      />
    </Box>
  );
};

export default BackupPage;
