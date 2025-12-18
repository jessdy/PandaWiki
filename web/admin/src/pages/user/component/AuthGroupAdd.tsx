import { Box, Button, Stack, TextField } from '@mui/material';
import { FormItem } from '@/components/Form';
import { Modal, message } from '@ctzhian/ui';
import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { postApiV1UserAuthGroupCreate } from '@/request/User';

type AuthGroupAddProps = {
  refresh: () => void;
  kbId: string;
};

const AuthGroupAdd = ({ refresh, kbId }: AuthGroupAddProps) => {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const {
    control,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm({
    defaultValues: {
      name: '',
      position: 0,
      auth_ids: [] as number[],
    },
  });

  const onSubmit = async (data: any) => {
    if (!kbId) {
      message.error('请先选择知识库');
      return;
    }
    setLoading(true);
    try {
      await postApiV1UserAuthGroupCreate({
        name: data.name,
        kb_id: kbId,
        position: data.position || 0,
        auth_ids: data.auth_ids || [],
      });
      message.success('创建成功');
      setOpen(false);
      reset();
      refresh();
    } catch (err: any) {
      message.error(err?.message || '创建失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Button variant='contained' size='small' onClick={() => setOpen(true)}>
        添加用户组
      </Button>
      <Modal
        open={open}
        onClose={() => {
          setOpen(false);
          reset();
        }}
        title='添加用户组'
        width={500}
        footer={null}
      >
        <form onSubmit={handleSubmit(onSubmit)}>
          <Stack gap={2}>
            <FormItem label='用户组名称' required>
              <Controller
                name='name'
                control={control}
                rules={{ required: '请输入用户组名称' }}
                render={({ field }) => (
                  <TextField
                    {...field}
                    size='small'
                    placeholder='请输入用户组名称'
                    error={!!errors.name}
                    helperText={errors.name?.message as string}
                  />
                )}
              />
            </FormItem>
            <FormItem label='排序位置'>
              <Controller
                name='position'
                control={control}
                render={({ field }) => (
                  <TextField
                    {...field}
                    type='number'
                    size='small'
                    placeholder='0'
                    onChange={e => field.onChange(Number(e.target.value))}
                  />
                )}
              />
            </FormItem>
            <Stack
              direction='row'
              gap={2}
              justifyContent='flex-end'
              sx={{ mt: 2 }}
            >
              <Button
                onClick={() => {
                  setOpen(false);
                  reset();
                }}
              >
                取消
              </Button>
              <Button type='submit' variant='contained' disabled={loading}>
                确定
              </Button>
            </Stack>
          </Stack>
        </form>
      </Modal>
    </>
  );
};

export default AuthGroupAdd;
