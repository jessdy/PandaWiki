import {
  getApiV1ApprovalList,
  postApiV1ApprovalApprove,
  postApiV1ApprovalReject,
  UserApprovalItem,
} from '@/request/Approval';
import { ConstsUserRole } from '@/request/types';
import { useAppSelector } from '@/store';
import { Box, Button, Stack, TextField, Typography } from '@mui/material';
import { message, Modal, Table } from '@ctzhian/ui';
import { useEffect, useState } from 'react';

const statusLabel: Record<string, string> = {
  pending: '待审批',
  approved: '已通过',
  rejected: '已拒绝',
  failed: '执行失败',
};

const ApprovalPage = () => {
  const { user } = useAppSelector(state => state.config);
  const isSec = user.role === ConstsUserRole.UserRoleSecurityAdmin;
  const [list, setList] = useState<UserApprovalItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState(isSec ? 'pending' : '');
  const [loading, setLoading] = useState(false);
  const [comment, setComment] = useState('');

  const load = async (p = page) => {
    setLoading(true);
    try {
      const res = await getApiV1ApprovalList({
        page: p,
        per_page: 20,
        status: status || undefined,
      });
      setList(res?.list || []);
      setTotal(res?.total || 0);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load(1);
  }, [status]);

  const review = (item: UserApprovalItem, approve: boolean) => {
    Modal.confirm({
      title: approve ? '通过申请' : '拒绝申请',
      content: (
        <Stack gap={2}>
          <Typography>{item.summary}</Typography>
          <TextField
            fullWidth
            label='审批意见'
            defaultValue={comment}
            onChange={e => setComment(e.target.value)}
          />
        </Stack>
      ),
      onOk: async () => {
        if (approve) await postApiV1ApprovalApprove(item.id, { comment });
        else await postApiV1ApprovalReject(item.id, { comment });
        message.success(approve ? '已通过' : '已拒绝');
        setComment('');
        void load(page);
      },
    });
  };

  return (
    <Box sx={{ p: 2 }}>
      <Stack direction='row' justifyContent='space-between' mb={2}>
        <Typography variant='h6'>
          {isSec ? '安全审批' : '我的审批申请'}
        </Typography>
        <Stack direction='row' gap={1}>
          {['', 'pending', 'approved', 'rejected', 'failed'].map(s => (
            <Button
              key={s || 'all'}
              size='small'
              variant={status === s ? 'contained' : 'outlined'}
              onClick={() => {
                setStatus(s);
                setPage(1);
              }}
            >
              {s ? statusLabel[s] : '全部'}
            </Button>
          ))}
        </Stack>
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
          { title: '类型', dataIndex: 'type', width: 140 },
          { title: '摘要', dataIndex: 'summary' },
          { title: '申请人', dataIndex: 'requester_account', width: 120 },
          {
            title: '状态',
            dataIndex: 'status',
            width: 100,
            render: (v: string) => statusLabel[v] || v,
          },
          {
            title: '操作',
            dataIndex: 'id',
            width: 160,
            render: (_: unknown, row: UserApprovalItem) =>
              isSec && row.status === 'pending' ? (
                <Stack direction='row' gap={1}>
                  <Button size='small' onClick={() => review(row, true)}>
                    通过
                  </Button>
                  <Button
                    size='small'
                    color='error'
                    onClick={() => review(row, false)}
                  >
                    拒绝
                  </Button>
                </Stack>
              ) : (
                row.review_comment || row.error || '-'
              ),
          },
        ]}
      />
    </Box>
  );
};

export default ApprovalPage;
