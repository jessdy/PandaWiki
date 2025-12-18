import Card from '@/components/Card';
import { copyText, generatePassword } from '@/utils';
import { CheckCircle } from '@mui/icons-material';
import { Box, Button, Stack, TextField } from '@mui/material';
import { FormItem } from '@/components/Form';
import { Modal, message } from '@ctzhian/ui';
import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { postApiV1UserGuestCreate } from '@/request/User';

type UserAddProps = {
  refresh: () => void;
};

const UserAdd = ({ refresh }: UserAddProps) => {
  const [addUser, setAddUser] = useState(false);
  const [loading, setLoading] = useState(false);
  const [password, setPassword] = useState('');
  const {
    control,
    handleSubmit,
    formState: { errors },
    reset,
    setValue,
    watch,
  } = useForm({
    defaultValues: {
      account: '',
      password: '',
    },
  });

  const account = watch('account');

  const copyUserInfo = ({
    account,
    password,
  }: {
    account: string;
    password: string;
  }) => {
    copyText(`用户名: ${account}\n密码: ${password}`, () => {
      setPassword('');
      reset();
    });
  };

  const onSubmit = handleSubmit(data => {
    setLoading(true);
    const password = data.password || generatePassword();
    postApiV1UserGuestCreate({
      account: data.account,
      password,
      role: 'guest' as any,
    })
      .then(() => {
        setPassword(password);
        setAddUser(false);
        refresh();
      })
      .catch((err: any) => {
        message.error(err?.message || '创建失败');
      })
      .finally(() => {
        setLoading(false);
      });
  });

  return (
    <>
      <Button
        size='small'
        variant='outlined'
        onClick={() => {
          setAddUser(true);
        }}
      >
        添加新用户
      </Button>
      <Modal
        title={
          <Stack direction='row' alignItems='center' gap={1}>
            <CheckCircle sx={{ color: 'success.main' }} />
            新用户创建成功
          </Stack>
        }
        open={!!password}
        closable={false}
        cancelText='关闭'
        onCancel={() => {
          setPassword('');
          reset();
        }}
        okText='复制用户信息'
        okButtonProps={{
          sx: { minWidth: '120px' },
        }}
        onOk={() => copyUserInfo({ account, password })}
      >
        <Card sx={{ p: 2, fontSize: 14, bgcolor: 'background.paper3' }}>
          <Stack direction={'row'}>
            <Box sx={{ width: 80 }}>用户名</Box>
            <Box sx={{ fontWeight: 700 }}>{account}</Box>
          </Stack>
          <Stack direction={'row'} sx={{ mt: 1 }}>
            <Box sx={{ width: 80 }}>密码</Box>
            <Box sx={{ fontWeight: 700 }}>{password}</Box>
          </Stack>
        </Card>
      </Modal>
      <Modal
        title='添加新用户'
        open={addUser}
        onCancel={() => {
          setAddUser(false);
          reset();
        }}
        onOk={onSubmit}
        okButtonProps={{
          loading,
        }}
      >
        <FormItem label='用户名' required>
          <Controller
            control={control}
            name='account'
            rules={{
              required: {
                value: true,
                message: '用户名不能为空',
              },
            }}
            render={({ field }) => (
              <TextField
                {...field}
                fullWidth
                autoFocus
                placeholder='输入用户名'
                error={!!errors.account}
                helperText={errors.account?.message}
              />
            )}
          />
        </FormItem>

        <FormItem label='密码' sx={{ mt: 2 }}>
          <Stack direction={'row'} alignItems={'center'} gap={2}>
            <Controller
              control={control}
              name='password'
              rules={{
                minLength: {
                  value: 8,
                  message: '密码长度至少 8 位',
                },
              }}
              render={({ field }) => (
                <TextField
                  {...field}
                  fullWidth
                  size='small'
                  placeholder='留空将自动生成密码'
                  error={!!errors.password}
                  helperText={errors.password?.message || '留空将自动生成密码'}
                />
              )}
            />
            <Button
              size='small'
              variant='outlined'
              onClick={() => setValue('password', generatePassword())}
            >
              生成
            </Button>
          </Stack>
        </FormItem>
      </Modal>
    </>
  );
};

export default UserAdd;
