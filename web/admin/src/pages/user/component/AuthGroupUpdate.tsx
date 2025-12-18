import { Box, Button, Stack, TextField } from '@mui/material';
import { FormItem } from '@/components/Form';
import { Modal, message } from '@ctzhian/ui';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { putApiV1UserAuthGroupId } from '@/request/User';

interface AuthGroup {
  id: number;
  name: string;
  kb_id: string;
  parent_id?: number;
  position: number;
  auth_ids: number[];
}

type AuthGroupUpdateProps = {
  group: AuthGroup | null;
  refresh: () => void;
  onClose: () => void;
};

const AuthGroupUpdate = ({ group, refresh, onClose }: AuthGroupUpdateProps) => {
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

  useEffect(() => {
    if (group) {
      reset({
        name: group.name,
        position: group.position,
        auth_ids: group.auth_ids || [],
      });
    }
  }, [group, reset]);

  const onSubmit = async (data: any) => {
    if (!group) return;
    setLoading(true);
    try {
      await putApiV1UserAuthGroupId(group.id, {
        name: data.name,
        position: data.position || 0,
        auth_ids: data.auth_ids || [],
      });
      message.success('更新成功');
      onClose();
      refresh();
    } catch (err: any) {
      message.error(err?.message || '更新失败');
    } finally {
      setLoading(false);
    }
  };

  if (!group) return null;

  return (
    <Modal
      open={true}
      onClose={onClose}
      title='编辑用户组'
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
            <Button onClick={onClose}>取消</Button>
            <Button type='submit' variant='contained' disabled={loading}>
              确定
            </Button>
          </Stack>
        </Stack>
      </form>
    </Modal>
  );
};

export default AuthGroupUpdate;
