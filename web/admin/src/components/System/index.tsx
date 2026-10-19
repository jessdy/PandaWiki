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
import TestEnvConfig from './component/TestEnvConfig';
import CloudAuthConfig, { CloudAuthConfigRef } from './component/CloudAuthConfig';
import { getApiV1SystemTestEnv } from '@/request/TestEnv';

type SystemTab = 'model' | 'standard-terms' | 'test-env' | 'cloud-auth';

const System = () => {
  const { user, modelList, isCreateWikiModalOpen } = useAppSelector(
    state => state.config,
  );
  const [open, setOpen] = useState(false);
  const dispatch = useAppDispatch();
  const modelConfigRef = useRef<ModelConfigRef>(null);
  const standardTermsConfigRef = useRef<StandardTermsConfigRef>(null);
  const cloudAuthConfigRef = useRef<CloudAuthConfigRef>(null);
  const [activeTab, setActiveTab] = useState<SystemTab>('model');
  const [testEnvEnabled, setTestEnvEnabled] = useState(false);
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
    testEnvOn = testEnvEnabled,
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

    // 测试环境只要求对话模型；关闭后仍要求向量模型和重排模型。
    const status = testEnvOn ? !!chat : !!(chat && embedding && rerank);
    dispatch(setModelStatus(status));
  };

  useEffect(() => {
    if (user.role !== 'admin') return;
    getApiV1SystemTestEnv()
      .then(res => setTestEnvEnabled(!!res?.enabled))
      .catch(() => setTestEnvEnabled(false));
  }, [user.role]);

  useEffect(() => {
    if (modelList) {
      handleModelList(modelList, testEnvEnabled);
    }
    // handleModelList 读取当前模型列表，随测试环境开关一起刷新状态。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [modelList, testEnvEnabled]);

  useEffect(() => {
    if (isCreateWikiModalOpen) {
      setOpen(false);
    }
  }, [isCreateWikiModalOpen]);

  const currentConfigHasChanges = () => {
    if (activeTab === 'model') {
      return modelConfigRef.current?.hasUnsavedChanges() ?? false;
    }
    if (activeTab === 'standard-terms') {
      return standardTermsConfigRef.current?.hasUnsavedChanges() ?? false;
    }
    if (activeTab === 'cloud-auth') {
      return cloudAuthConfigRef.current?.hasUnsavedChanges() ?? false;
    }
    return false;
  };

  const closeCurrentConfig = () => {
    if (activeTab === 'model') {
      if (modelConfigRef.current) modelConfigRef.current.handleClose();
      else setOpen(false);
    } else if (activeTab === 'standard-terms') {
      if (standardTermsConfigRef.current)
        standardTermsConfigRef.current.handleClose();
      else setOpen(false);
    } else if (activeTab === 'cloud-auth') {
      if (cloudAuthConfigRef.current) cloudAuthConfigRef.current.handleClose();
      else setOpen(false);
    } else {
      setOpen(false);
    }
  };

  const switchTab = (nextTab: SystemTab) => {
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
          onChange={(_, value: SystemTab) => switchTab(value)}
          sx={{ mb: 3, borderBottom: '1px solid', borderColor: 'divider' }}
        >
          <Tab value='model' label='模型配置' />
          <Tab value='standard-terms' label='标准术语' />
          <Tab value='test-env' label='测试环境' />
          <Tab value='cloud-auth' label='平台地址' />
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
        ) : activeTab === 'standard-terms' ? (
          <StandardTermsConfig
            ref={standardTermsConfigRef}
            onCloseModal={() => setOpen(false)}
          />
        ) : activeTab === 'test-env' ? (
          <TestEnvConfig onEnabledChange={setTestEnvEnabled} />
        ) : (
          <CloudAuthConfig
            ref={cloudAuthConfigRef}
            onCloseModal={() => setOpen(false)}
          />
        )}
      </Modal>
    </>
  );
};
export default System;
