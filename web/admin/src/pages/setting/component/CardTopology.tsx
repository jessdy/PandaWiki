import { putApiV1App } from '@/request/App';
import { DomainAppDetailResp, DomainTopologySettings } from '@/request/types';
import { useAppSelector } from '@/store';
import { message } from '@ctzhian/ui';
import {
  FormControlLabel,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { FormItem, SettingCardItem } from './Common';

interface CardTopologyProps {
  id: string;
  data: DomainAppDetailResp;
  refresh: (value: DomainTopologySettings) => void;
}

const CardTopology = ({ id, data, refresh }: CardTopologyProps) => {
  const [isEdit, setIsEdit] = useState(false);
  const { kb_id } = useAppSelector(state => state.config);
  const { control, handleSubmit, setValue, watch } =
    useForm<DomainTopologySettings>({
      defaultValues: {
        enabled: false,
        title: '',
        description: '',
      },
    });

  const enabled = watch('enabled');

  useEffect(() => {
    const topology = data.settings?.topology_settings;
    setValue('enabled', !!topology?.enabled);
    setValue('title', topology?.title || '');
    setValue('description', topology?.description || '');
  }, [data, setValue]);

  const onSubmit = (value: DomainTopologySettings) => {
    const topology_settings: DomainTopologySettings = {
      enabled: !!value.enabled,
      title: (value.title || '').trim(),
      description: (value.description || '').trim(),
    };
    putApiV1App(
      { id },
      {
        kb_id,
        settings: {
          ...data.settings,
          topology_settings,
        },
      },
    ).then(() => {
      refresh(topology_settings);
      message.success('保存成功');
      setIsEdit(false);
    });
  };

  return (
    <SettingCardItem
      title='知识拓扑图'
      isEdit={isEdit}
      onSubmit={handleSubmit(onSubmit)}
    >
      <FormItem label='展示区块' sx={{ alignItems: 'flex-start' }}>
        <Stack sx={{ flex: 1 }}>
          <Controller
            control={control}
            name='enabled'
            render={({ field }) => (
              <FormControlLabel
                control={
                  <Switch
                    size='small'
                    checked={!!field.value}
                    onChange={e => {
                      field.onChange(e.target.checked);
                      setIsEdit(true);
                    }}
                  />
                }
                label='在自定义首页展示「知识拓扑图」区块'
              />
            )}
          />
          <Typography variant='caption' color='text.secondary'>
            开启后，首页会展示一个交互式拓扑图，逐层展开被勾选「在首页拓扑图中展示」的目录与文档。仅在首页类型为「自定义」时生效。
          </Typography>
        </Stack>
      </FormItem>
      {enabled && (
        <>
          <FormItem label='区块标题'>
            <Controller
              control={control}
              name='title'
              render={({ field }) => (
                <TextField
                  {...field}
                  fullWidth
                  size='small'
                  placeholder='知识拓扑图'
                  onChange={e => {
                    field.onChange(e.target.value);
                    setIsEdit(true);
                  }}
                />
              )}
            />
          </FormItem>
          <FormItem label='区块描述'>
            <Controller
              control={control}
              name='description'
              render={({ field }) => (
                <TextField
                  {...field}
                  fullWidth
                  size='small'
                  placeholder='点击节点可逐层展开知识结构'
                  onChange={e => {
                    field.onChange(e.target.value);
                    setIsEdit(true);
                  }}
                />
              )}
            />
          </FormItem>
        </>
      )}
    </SettingCardItem>
  );
};

export default CardTopology;
