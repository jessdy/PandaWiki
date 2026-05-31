import Card from '@/components/Card';
import {
  ConsultInquiryListItem,
  ConsultStatus,
  deleteApiV1Consult,
  getApiV1ConsultList,
} from '@/request/Consult';
import { tableSx } from '@/constant/styles';
import {
  Box,
  Button,
  Chip,
  Stack,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material';
import { Ellipsis, message, Modal, Table } from '@ctzhian/ui';
import { ColumnsType } from '@ctzhian/ui/dist/Table';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useState } from 'react';
import ConsultDetailDrawer from './ConsultDetailDrawer';

const STATUS_LABELS: Record<
  ConsultStatus,
  { label: string; color: 'default' | 'warning' | 'info' | 'success' }
> = {
  pending: { label: '待处理', color: 'warning' },
  processing: { label: '处理中', color: 'info' },
  replied: { label: '已回复', color: 'success' },
  closed: { label: '已关闭', color: 'default' },
};

type StatusFilter = '' | ConsultStatus;

const Consult = () => {
  const [filter, setFilter] = useState<StatusFilter>('');
  const [keyword, setKeyword] = useState('');
  const [keywordInput, setKeywordInput] = useState('');
  const [data, setData] = useState<ConsultInquiryListItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [selected, setSelected] = useState<number[]>([]);

  const [detailId, setDetailId] = useState<number | null>(null);

  useEffect(() => {
    setPage(1);
  }, [filter, keyword]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const body = (await getApiV1ConsultList({
        page,
        per_page: pageSize,
        ...(filter ? { status: filter } : {}),
        ...(keyword ? { keyword } : {}),
      })) as { data?: ConsultInquiryListItem[]; total?: number };
      setData(body?.data ?? []);
      setTotal(body?.total ?? 0);
      setSelected([]);
      window.dispatchEvent(new Event('consult-open-count-changed'));
    } catch (e: unknown) {
      message.error((e as { message?: string })?.message || '加载失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, filter, keyword]);

  useEffect(() => {
    void load();
  }, [load]);

  const onBatchDelete = () => {
    if (selected.length === 0) return;
    Modal.confirm({
      title: '确认删除？',
      content: `将删除 ${selected.length} 条咨询及其完整对话记录，不可恢复。`,
      onOk: async () => {
        try {
          await deleteApiV1Consult({ ids: selected.join(',') });
          message.success('已删除');
          void load();
        } catch (e: unknown) {
          message.error((e as { message?: string })?.message || '删除失败');
        }
      },
    });
  };

  const columns: ColumnsType<ConsultInquiryListItem> = [
    {
      dataIndex: 'title',
      title: '咨询内容',
      render: (_: unknown, row: ConsultInquiryListItem) => (
        <Ellipsis sx={{ maxWidth: 360 }}>
          {row.title || row.content.slice(0, 60) || '-'}
        </Ellipsis>
      ),
    },
    {
      dataIndex: 'submitter_name',
      title: '提问人',
      width: 120,
      render: (_: unknown, row: ConsultInquiryListItem) => (
        <Ellipsis sx={{ maxWidth: 100 }}>
          {row.submitter_name || row.user_id || '-'}
        </Ellipsis>
      ),
    },
    {
      dataIndex: 'contact',
      title: '联系方式',
      width: 180,
      render: (v: string) => (
        <Ellipsis sx={{ maxWidth: 160 }}>{v || '-'}</Ellipsis>
      ),
    },
    {
      dataIndex: 'status',
      title: '状态',
      width: 90,
      render: (v: ConsultStatus) => {
        const s = STATUS_LABELS[v] || STATUS_LABELS.pending;
        return <Chip label={s.label} color={s.color} size='small' />;
      },
    },
    {
      dataIndex: 'reply_count',
      title: '消息数',
      width: 80,
      // 消息总条数 = 1 首条 + reply_count 后续
      render: (v: number) => 1 + (v || 0),
    },
    {
      dataIndex: 'last_message_at',
      title: '最近活跃',
      width: 160,
      render: (v: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      dataIndex: 'created_at',
      title: '创建时间',
      width: 160,
      render: (v: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      dataIndex: 'op',
      title: '操作',
      width: 120,
      render: (_: unknown, row: ConsultInquiryListItem) => (
        <Button size='small' onClick={() => setDetailId(row.id)}>
          查看详情
        </Button>
      ),
    },
  ];

  return (
    <Card>
      <Stack
        direction='row'
        alignItems='center'
        justifyContent='space-between'
        sx={{ p: 2, gap: 2, flexWrap: 'wrap' }}
      >
        <Stack direction='row' alignItems='center' gap={1.5} flexWrap='wrap'>
          <ToggleButtonGroup
            size='small'
            value={filter}
            exclusive
            onChange={(_, v) => v !== null && setFilter(v as StatusFilter)}
          >
            <ToggleButton value=''>全部</ToggleButton>
            <ToggleButton value='pending'>待处理</ToggleButton>
            <ToggleButton value='processing'>处理中</ToggleButton>
            <ToggleButton value='replied'>已回复</ToggleButton>
            <ToggleButton value='closed'>已关闭</ToggleButton>
          </ToggleButtonGroup>
          <TextField
            size='small'
            placeholder='搜索内容 / 联系方式 / 标题'
            value={keywordInput}
            onChange={e => setKeywordInput(e.target.value)}
            onKeyDown={e => {
              if (e.key === 'Enter') setKeyword(keywordInput.trim());
            }}
            sx={{ minWidth: 260 }}
          />
          <Button
            size='small'
            variant='outlined'
            onClick={() => setKeyword(keywordInput.trim())}
          >
            搜索
          </Button>
          {keyword && (
            <Button
              size='small'
              onClick={() => {
                setKeyword('');
                setKeywordInput('');
              }}
            >
              清空
            </Button>
          )}
        </Stack>
        <Stack direction='row' gap={1}>
          <Button
            size='small'
            color='error'
            variant='outlined'
            disabled={selected.length === 0}
            onClick={onBatchDelete}
          >
            批量删除 {selected.length > 0 ? `(${selected.length})` : ''}
          </Button>
        </Stack>
      </Stack>

      <Box sx={{ px: 2, pb: 2 }}>
        <Table
          rowKey='id'
          columns={columns}
          dataSource={data}
          size='small'
          sx={tableSx}
          loading={loading}
          rowSelection={{
            selectedRowKeys: selected,
            // @ctzhian/ui 的 Key 类型含 bigint，业务里 id 都是 int64 序列化为 number，
            // 这里直接转换并丢掉 bigint 情况（实际不会出现）
            onChange: (keys: React.Key[]) =>
              setSelected(keys.map(k => Number(k))),
          }}
          pagination={{
            page,
            pageSize,
            total,
            onChange: (p: number, ps: number) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Box>

      <ConsultDetailDrawer
        id={detailId}
        onClose={() => setDetailId(null)}
        onChanged={() => void load()}
      />
    </Card>
  );
};

export default Consult;
