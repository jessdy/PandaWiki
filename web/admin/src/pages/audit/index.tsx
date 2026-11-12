import {
  AuditLogItem,
  getApiV1AuditLogs,
  getApiV1AuditStorage,
  getApiV1AuditVerify,
  postApiV1AuditExport,
  putApiV1AuditStorage,
} from '@/request/Audit';
import {
  Box,
  Button,
  Stack,
  TextField,
  MenuItem,
  Typography,
} from '@mui/material';
import { message, Modal, Table } from '@ctzhian/ui';
import { useEffect, useState } from 'react';

const AuditPage = () => {
  const [list, setList] = useState<AuditLogItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState('');
  const [result, setResult] = useState('');
  const [loading, setLoading] = useState(false);
  const [storageText, setStorageText] = useState('');
  const [exportPassword, setExportPassword] = useState('');
  const [exportOpen, setExportOpen] = useState(false);
  const [limitMb, setLimitMb] = useState(10240);
  const [alertPercent, setAlertPercent] = useState(80);

  const load = async (p = page) => {
    setLoading(true);
    try {
      const res = await getApiV1AuditLogs({
        page: p,
        per_page: 20,
        keyword: keyword || undefined,
        result: result || undefined,
      });
      setList(res?.list || []);
      setTotal(res?.total || 0);
    } finally {
      setLoading(false);
    }
  };

  const loadStorage = async () => {
    const res = await getApiV1AuditStorage();
    const used = ((res.table_bytes || 0) / 1024 / 1024).toFixed(1);
    const limit = ((res.limit_bytes || 0) / 1024 / 1024).toFixed(0);
    setStorageText(
      `记录 ${res.rows} 条，占用 ${used} MB / ${limit} MB（${res.used_percent.toFixed(1)}%）` +
        (res.alert ? ' 【已超告警阈值】' : ''),
    );
  };

  useEffect(() => {
    void load(1);
    void loadStorage();
  }, []);

  return (
    <Box sx={{ p: 2 }}>
      <Stack
        direction='row'
        justifyContent='space-between'
        alignItems='center'
        mb={2}
      >
        <Typography variant='h6'>安全审计日志</Typography>
        <Typography variant='body2' color='text.secondary'>
          {storageText}
        </Typography>
      </Stack>
      <Stack direction='row' gap={1} mb={2} flexWrap='wrap'>
        <TextField
          size='small'
          label='关键词'
          value={keyword}
          onChange={e => setKeyword(e.target.value)}
        />
        <TextField
          select
          size='small'
          label='结果'
          value={result}
          onChange={e => setResult(e.target.value)}
          sx={{ minWidth: 120 }}
        >
          <MenuItem value=''>全部</MenuItem>
          <MenuItem value='success'>成功</MenuItem>
          <MenuItem value='failure'>失败</MenuItem>
        </TextField>
        <Button
          variant='contained'
          onClick={() => {
            setPage(1);
            void load(1);
          }}
        >
          查询
        </Button>
        <Button
          variant='outlined'
          onClick={async () => {
            const res = await getApiV1AuditVerify();
            if (res.valid) message.success(`哈希链完整，共 ${res.total} 条`);
            else message.error(`哈希链异常 #${res.broken_id}: ${res.reason}`);
          }}
        >
          校验完整性
        </Button>
        <Button variant='outlined' onClick={() => setExportOpen(true)}>
          导出（加密+签名）
        </Button>
        <Button
          variant='outlined'
          onClick={async () => {
            await putApiV1AuditStorage({
              limit_mb: limitMb,
              alert_percent: alertPercent,
            });
            message.success('存储告警阈值已保存');
            void loadStorage();
          }}
        >
          保存告警阈值
        </Button>
        <TextField
          size='small'
          type='number'
          label='容量上限(MB)'
          value={limitMb}
          onChange={e => setLimitMb(Number(e.target.value) || 10240)}
          sx={{ width: 140 }}
        />
        <TextField
          size='small'
          type='number'
          label='告警(%)'
          value={alertPercent}
          onChange={e => setAlertPercent(Number(e.target.value) || 80)}
          sx={{ width: 100 }}
        />
      </Stack>
      <Table
        loading={loading}
        dataSource={list}
        rowKey='id'
        pagination={{
          page,
          pageSize: 20,
          total,
          onChange: (p: number) => {
            setPage(p);
            void load(p);
          },
        }}
        columns={[
          { title: '时间', dataIndex: 'created_at', width: 180 },
          { title: '账号', dataIndex: 'account', width: 120 },
          { title: 'IP', dataIndex: 'ip', width: 120 },
          { title: '事件', dataIndex: 'event', width: 160 },
          { title: '客体', dataIndex: 'object_id', width: 140 },
          { title: '结果', dataIndex: 'result', width: 80 },
          { title: '路径', dataIndex: 'path' },
        ]}
      />
      <Modal
        title='导出审计日志'
        open={exportOpen}
        onCancel={() => setExportOpen(false)}
        onOk={async () => {
          if (exportPassword.length < 8) {
            message.error('导出口令至少 8 位');
            return;
          }
          const file = (await postApiV1AuditExport({
            password: exportPassword,
            keyword: keyword || undefined,
            result: result || undefined,
          })) as unknown as import('@/request/Audit').AuditExportFile;
          const blob = new Blob([JSON.stringify(file, null, 2)], {
            type: 'application/json',
          });
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = `audit-export-${Date.now()}.json`;
          a.click();
          URL.revokeObjectURL(url);
          message.success(`已导出 ${file.count} 条`);
          setExportOpen(false);
          setExportPassword('');
        }}
      >
        <Typography variant='body2' sx={{ mb: 2 }}>
          导出会生成 SM4 加密、HMAC-SM3 签名的文件，请设置导出口令并妥善保管。
        </Typography>
        <TextField
          fullWidth
          type='password'
          label='导出口令'
          value={exportPassword}
          onChange={e => setExportPassword(e.target.value)}
        />
      </Modal>
    </Box>
  );
};

export default AuditPage;
