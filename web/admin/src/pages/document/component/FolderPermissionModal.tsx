import { Form, FormItem } from '@/pages/setting/component/Common';
import { getApiV1NodeDetail, putApiV1NodeDetail } from '@/request/Node';
import { patchApiV1NodePermissionEdit } from '@/request/NodePermission';
import { getApiProV1AuthGroupList } from '@/request/pro/AuthGroup';
import type { GithubComChaitinPandaWikiProApiAuthV1AuthGroupListItem } from '@/request/pro/types';
import { ConstsNodeAccessPerm } from '@/request/types';
import { Modal, message } from '@ctzhian/ui';
import {
  Autocomplete,
  Checkbox,
  FormControlLabel,
  Radio,
  RadioGroup,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';

export interface FolderPermissionModalProps {
  open: boolean;
  onCancel: () => void;
  onSuccess: () => void;
  nodeId: string;
  nodeName?: string;
  kbId: string;
}

type FormValues = {
  perm: ConstsNodeAccessPerm | null;
  groups: GithubComChaitinPandaWikiProApiAuthV1AuthGroupListItem[];
  apply_children: string;
  work_mode_directory: boolean;
  show_in_topology: boolean;
};

const FolderPermissionModal = ({
  open,
  onCancel,
  onSuccess,
  nodeId,
  nodeName,
  kbId,
}: FolderPermissionModalProps) => {
  const [userGroups, setUserGroups] = useState<
    GithubComChaitinPandaWikiProApiAuthV1AuthGroupListItem[]
  >([]);
  const [loading, setLoading] = useState(false);

  const { control, handleSubmit, setValue, reset, watch } = useForm<FormValues>(
    {
      defaultValues: {
        perm: ConstsNodeAccessPerm.NodeAccessPermOpen,
        groups: [],
        apply_children: 'true',
        work_mode_directory: false,
        show_in_topology: false,
      },
    },
  );

  const watchPerm = watch('perm');

  useEffect(() => {
    if (!open || !nodeId || !kbId) return;
    getApiProV1AuthGroupList({ kb_id: kbId, page: 1, per_page: 9999 })
      .then(res => {
        if (res?.list) setUserGroups(res.list);
        else setUserGroups([]);
      })
      .catch(() => setUserGroups([]));

    setValue('perm', ConstsNodeAccessPerm.NodeAccessPermOpen);
    setValue('groups', []);
    setValue('apply_children', 'true');
    setValue('work_mode_directory', false);
    setValue('show_in_topology', false);

    getApiV1NodeDetail({ kb_id: kbId, id: nodeId })
      .then(res => {
        setValue('work_mode_directory', !!res.meta?.work_mode_directory);
        setValue('show_in_topology', !!res.meta?.show_in_topology);
      })
      .catch(() => {
        setValue('work_mode_directory', false);
        setValue('show_in_topology', false);
      });
  }, [open, nodeId, kbId, setValue]);

  useEffect(() => {
    if (!open) reset();
  }, [open, reset]);

  const onSubmit = handleSubmit(values => {
    setLoading(true);
    const permValue = values.perm ?? ConstsNodeAccessPerm.NodeAccessPermOpen;
    const groupIds =
      permValue === ConstsNodeAccessPerm.NodeAccessPermPartial
        ? (values.groups || []).map(g => g.id!)
        : [];

    Promise.all([
      patchApiV1NodePermissionEdit({
        kb_id: kbId,
        ids: [nodeId],
        permissions: {
          answerable: permValue,
          visitable: permValue,
          visible: permValue,
        },
        answerable_groups: groupIds,
        visitable_groups: groupIds,
        visible_groups: groupIds,
        apply_children: values.apply_children === 'true',
      }),
      putApiV1NodeDetail({
        id: nodeId,
        kb_id: kbId,
        work_mode_directory: values.work_mode_directory,
        show_in_topology: values.show_in_topology,
      }),
    ])
      .then(() => {
        message.success('保存成功');
        onSuccess();
        onCancel();
      })
      .finally(() => setLoading(false));
  });

  return (
    <Modal
      title={`编辑开放权限${nodeName ? `：${nodeName}` : ''}`}
      open={open}
      onCancel={onCancel}
      width={480}
      okButtonProps={{ loading: loading }}
      onOk={onSubmit}
    >
      <Form labelWidth={100} gap={3}>
        <FormItem label='开放权限' required>
          <Controller
            control={control}
            name='perm'
            render={({ field }) => (
              <RadioGroup row {...field} sx={{ gap: 2 }}>
                <FormControlLabel
                  value={ConstsNodeAccessPerm.NodeAccessPermOpen}
                  control={<Radio size='small' />}
                  label='完全开放'
                />
                <FormControlLabel
                  value={ConstsNodeAccessPerm.NodeAccessPermPartial}
                  control={<Radio size='small' />}
                  label='部分开放'
                />
                <FormControlLabel
                  value={ConstsNodeAccessPerm.NodeAccessPermClosed}
                  control={<Radio size='small' />}
                  label='完全禁止'
                />
              </RadioGroup>
            )}
          />
        </FormItem>
        {watchPerm === ConstsNodeAccessPerm.NodeAccessPermPartial && (
          <FormItem label='允许的用户组'>
            <Controller
              control={control}
              name='groups'
              render={({ field }) => (
                <Autocomplete
                  {...field}
                  fullWidth
                  multiple
                  options={userGroups}
                  getOptionLabel={option => option.path ?? option.name ?? ''}
                  onChange={(_, value) => field.onChange(value)}
                  isOptionEqualToValue={(option, value) =>
                    option.id === value.id
                  }
                  renderInput={params => (
                    <TextField
                      {...params}
                      size='small'
                      placeholder='选择允许的用户组'
                    />
                  )}
                />
              )}
            />
          </FormItem>
        )}
        <FormItem label='实战模式问答'>
          <Stack spacing={0.5}>
            <Controller
              control={control}
              name='work_mode_directory'
              render={({ field }) => (
                <FormControlLabel
                  control={
                    <Checkbox
                      size='small'
                      checked={field.value}
                      onChange={(_, c) => field.onChange(c)}
                    />
                  }
                  label='将本目录及子目录纳入前台「实战模式」检索范围'
                />
              )}
            />
            <Typography variant='caption' color='text.secondary'>
              未勾选任何目录时，实战模式仍检索全库；可多选文件夹分别圈定范围。
            </Typography>
          </Stack>
        </FormItem>
        <FormItem label='首页拓扑图'>
          <Stack spacing={0.5}>
            <Controller
              control={control}
              name='show_in_topology'
              render={({ field }) => (
                <FormControlLabel
                  control={
                    <Checkbox
                      size='small'
                      checked={field.value}
                      onChange={(_, c) => field.onChange(c)}
                    />
                  }
                  label='在前台首页「知识拓扑图」中展示该目录'
                />
              )}
            />
            <Typography variant='caption' color='text.secondary'>
              开启后，该目录会作为节点出现在首页拓扑图中；需在「门户网站 →
              知识拓扑图」中开启拓扑图区块后前台才会展示。
            </Typography>
          </Stack>
        </FormItem>
        <FormItem label='应用范围' required>
          <Controller
            control={control}
            name='apply_children'
            render={({ field }) => (
              <RadioGroup {...field}>
                <FormControlLabel
                  value='false'
                  control={<Radio size='small' />}
                  label='仅当前目录'
                />
                <FormControlLabel
                  value='true'
                  control={<Radio size='small' />}
                  label='同时应用到子目录及目录下的文件'
                />
              </RadioGroup>
            )}
          />
        </FormItem>
      </Form>
    </Modal>
  );
};

export default FolderPermissionModal;
