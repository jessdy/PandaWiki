import NoData from '@/assets/images/nodata.png';
import Card from '@/components/Card';
import { tableSx } from '@/constant/styles';
import { V1UserListItemResp } from '@/request/types';
import { Table } from '@ctzhian/ui';
import { ColumnType } from '@ctzhian/ui/dist/Table';
import { Box, Button, Stack, Tabs, Tab } from '@mui/material';
import dayjs from 'dayjs';
import { useEffect, useState } from 'react';
import UserAdd from './component/UserAdd';
import UserDelete from './component/UserDelete';
import UserUpdate from './component/UserUpdate';
import AuthGroupManage from './component/AuthGroupManage';
import { getApiV1UserGuestList } from '@/request/User';

const User = () => {
  const [tabValue, setTabValue] = useState(0);
  const [loading, setLoading] = useState(false);
  const [userList, setUserList] = useState<V1UserListItemResp[]>([]);
  const [curUser, setCurUser] = useState<V1UserListItemResp | null>(null);
  const [curType, setCurType] = useState<'delete' | 'update' | null>(null);

  const columns: ColumnType<V1UserListItemResp>[] = [
    {
      title: '用户名',
      dataIndex: 'account',
      render: (text: string) => <Box>{text}</Box>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      render: (text: string) => (
        <Box>{text ? dayjs(text).format('YYYY-MM-DD HH:mm:ss') : '-'}</Box>
      ),
    },
    {
      title: '上次使用时间',
      dataIndex: 'last_access',
      render: (text: string) => (
        <Box>{text ? dayjs(text).format('YYYY-MM-DD HH:mm:ss') : '-'}</Box>
      ),
    },
    {
      title: '',
      dataIndex: 'action',
      width: 140,
      render: (_, record) => (
        <Stack direction={'row'} gap={2}>
          <Button
            size='small'
            sx={{ p: 0, minWidth: 'auto' }}
            color='primary'
            onClick={() => {
              setCurUser(record);
              setCurType('update');
            }}
          >
            编辑
          </Button>
          <Button
            size='small'
            color='error'
            sx={{ p: 0, minWidth: 'auto' }}
            onClick={() => {
              setCurUser(record);
              setCurType('delete');
            }}
          >
            删除
          </Button>
        </Stack>
      ),
    },
  ];

  const getData = () => {
    setLoading(true);
    getApiV1UserGuestList()
      .then(
        (data: {
          data?: { users?: V1UserListItemResp[] };
          users?: V1UserListItemResp[];
        }) => {
          const res = data?.data?.users || data?.users || [];
          setUserList(res);
        },
      )
      .catch((err: Error) => {
        console.error('获取用户列表失败:', err);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    getData();
  }, []);

  return (
    <Card
      sx={{
        flex: 1,
        py: 2,
        overflow: 'hidden',
        overflowY: 'auto',
        border: '1px solid',
        borderColor: 'divider',
      }}
    >
      <Stack
        direction={'row'}
        justifyContent={'space-between'}
        alignItems={'center'}
        sx={{ mb: 2, px: 2 }}
      >
        <Box sx={{ fontSize: 14, lineHeight: '24px', fontWeight: 'bold' }}>
          用户管理
        </Box>
        {tabValue === 0 && <UserAdd refresh={getData} />}
      </Stack>
      <Tabs
        value={tabValue}
        onChange={(_, newValue) => setTabValue(newValue)}
        sx={{ px: 2, mb: 2 }}
      >
        <Tab label='用户' />
        <Tab label='用户组' />
      </Tabs>
      {tabValue === 0 && (
        <>
          <Table
            columns={columns}
            dataSource={userList}
            rowKey='id'
            size='small'
            updateScrollTop={false}
            height='calc(100vh - 200px)'
            sx={{ overflow: 'hidden', ...tableSx }}
            pagination={false}
            renderEmpty={
              loading ? (
                <Box></Box>
              ) : (
                <Stack alignItems={'center'}>
                  <img src={NoData} width={150} />
                  <Box
                    sx={{
                      fontSize: 12,
                      lineHeight: '20px',
                      color: 'text.tertiary',
                    }}
                  >
                    暂无数据
                  </Box>
                </Stack>
              )
            }
          />
          {curUser && curType === 'update' && (
            <UserUpdate
              user={curUser}
              refresh={getData}
              onClose={() => {
                setCurType(null);
                setCurUser(null);
              }}
            />
          )}
          <UserDelete
            open={!!curUser && curType === 'delete'}
            onClose={() => {
              setCurType(null);
              setCurUser(null);
            }}
            user={curUser}
            refresh={getData}
          />
        </>
      )}
      {tabValue === 1 && <AuthGroupManage />}
    </Card>
  );
};

export default User;
