import { Button, Stack } from '@mui/material';
import { Modal, message } from '@ctzhian/ui';
import { useState } from 'react';
import { deleteApiV1UserAuthGroupId } from '@/request/User';

interface AuthGroup {
  id: number;
  name: string;
}

type AuthGroupDeleteProps = {
  open: boolean;
  group: AuthGroup | null;
  refresh: () => void;
  onClose: () => void;
};

const AuthGroupDelete = ({
  open,
  group,
  refresh,
  onClose,
}: AuthGroupDeleteProps) => {
  const [loading, setLoading] = useState(false);

  const handleDelete = async () => {
    if (!group) return;
    setLoading(true);
    try {
      await deleteApiV1UserAuthGroupId(group.id);
      message.success('删除成功');
      onClose();
      refresh();
    } catch (err: any) {
      message.error(err?.message || '删除失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title='删除用户组'
      width={400}
      footer={
        <Stack direction='row' gap={2} justifyContent='flex-end'>
          <Button onClick={onClose}>取消</Button>
          <Button
            variant='contained'
            color='error'
            onClick={handleDelete}
            disabled={loading}
          >
            确定
          </Button>
        </Stack>
      }
    >
      确定要删除用户组 "{group?.name}" 吗？
    </Modal>
  );
};

export default AuthGroupDelete;
