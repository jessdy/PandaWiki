import { getApiV1ModelList } from '@/request/Model';
import { GithubComChaitinPandaWikiDomainModelListItem } from '@/request/types';
import { useAppDispatch, useAppSelector } from '@/store';
import { setModelList, setModelStatus } from '@/store/slices/config';
import { Modal } from '@ctzhian/ui';
import { IconAChilunshezhisheding } from '@panda-wiki/icons';
import { Box, Button, Tab, Tabs } from '@mui/material';
import { useEffect, useState, useRef } from 'react';

import ModelConfig, { ModelConfigRef } from './component/ModelConfig';
import StandardTermsConfig, {
  StandardTermsConfigRef,
} from './component/StandardTermsConfig';

const System = () => {
  const { user, modelList, isCreateWikiModalOpen } = useAppSelector(
    state => state.config,
  );
  const [open, setOpen] = useState(false);
  const dispatch = useAppDispatch();
  const modelConfigRef = useRef<ModelConfigRef>(null);
  const standardTermsConfigRef = useRef<StandardTermsConfigRef>(null);
  const [activeTab, setActiveTab] = useState<'model' | 'standard-terms'>(
    'model',
  );
  const [chatModelData, setChatModelData] =
    useState<GithubComChaitinPandaWikiDomainModelListItem | null>(null);
  const [embeddingModelData, setEmbeddingModelData] =
    useState<GithubComChaitinPandaWikiDomainModelListItem | null>(null);
  const [rerankModelData, setRerankModelData] =
    useState<GithubComChaitinPandaWikiDomainModelListItem | null>(null);
  const [analysisModelData, setAnalysisModelData] =
    useState<GithubComChaitinPandaWikiDomainModelListItem | null>(null);
  const [analysisVLModelData, setAnalysisVLModelData] =
    useState<GithubComChaitinPandaWikiDomainModelListItem | null>(null);

  const getModelList = () => {
    getApiV1ModelList().then(res => {
      dispatch(
        setModelList(res as GithubComChaitinPandaWikiDomainModelListItem[]),
      );
    });
  };

  const handleModelList = (
    list: GithubComChaitinPandaWikiDomainModelListItem[],
  ) => {
    const chat = list.find(it => it.type === 'chat') || null;
    const embedding = list.find(it => it.type === 'embedding') || null;
    const rerank = list.find(it => it.type === 'rerank') || null;
    const analysis = list.find(it => it.type === 'analysis') || null;
    const analysisVL = list.find(it => it.type === 'analysis-vl') || null;
    setChatModelData(chat);
    setEmbeddingModelData(embedding);
    setRerankModelData(rerank);
    setAnalysisModelData(analysis);
    setAnalysisVLModelData(analysisVL);

    // 检查模型配置状态
    const status = !!(chat && embedding && rerank);
    dispatch(setModelStatus(status));
  };

  useEffect(() => {
    if (modelList) {
      handleModelList(modelList);
    }
  }, [modelList]);

  useEffect(() => {
    if (isCreateWikiModalOpen) {
      setOpen(false);
    }
  }, [isCreateWikiModalOpen]);

  const currentConfigHasChanges = () =>
    activeTab === 'model'
      ? (modelConfigRef.current?.hasUnsavedChanges() ?? false)
      : (standardTermsConfigRef.current?.hasUnsavedChanges() ?? false);

  const closeCurrentConfig = () => {
    if (activeTab === 'model') {
      if (modelConfigRef.current) modelConfigRef.current.handleClose();
      else setOpen(false);
    } else {
      if (standardTermsConfigRef.current)
        standardTermsConfigRef.current.handleClose();
      else setOpen(false);
    }
  };

  const switchTab = (nextTab: 'model' | 'standard-terms') => {
    if (nextTab === activeTab) return;
    if (!currentConfigHasChanges()) {
      setActiveTab(nextTab);
      return;
    }
    Modal.confirm({
      title: '有未保存的设置',
      content: '切换后当前修改将丢失，是否继续？',
      okText: '确认切换',
      cancelText: '继续编辑',
      onOk: () => setActiveTab(nextTab),
    });
  };

  return (
    <>
      <Box sx={{ position: 'relative' }}>
        {user.role === 'admin' && (
          <Button
            size='small'
            variant='outlined'
            startIcon={<IconAChilunshezhisheding />}
            onClick={() => setOpen(true)}
          >
            系统配置
          </Button>
        )}
      </Box>
      <Modal
        title='系统配置'
        width={1100}
        open={open}
        disableEnforceFocus={true}
        footer={null}
        onCancel={closeCurrentConfig}
      >
        <Tabs
          value={activeTab}
          onChange={(_, value: 'model' | 'standard-terms') => switchTab(value)}
          sx={{ mb: 3, borderBottom: '1px solid', borderColor: 'divider' }}
        >
          <Tab value='model' label='模型配置' />
          <Tab value='standard-terms' label='标准术语' />
        </Tabs>
        {activeTab === 'model' ? (
          <ModelConfig
            ref={modelConfigRef}
            onCloseModal={() => setOpen(false)}
            chatModelData={chatModelData}
            embeddingModelData={embeddingModelData}
            rerankModelData={rerankModelData}
            analysisModelData={analysisModelData}
            analysisVLModelData={analysisVLModelData}
            getModelList={getModelList}
          />
        ) : (
          <StandardTermsConfig
            ref={standardTermsConfigRef}
            onCloseModal={() => setOpen(false)}
          />
        )}
      </Modal>
    </>
  );
};
export default System;
