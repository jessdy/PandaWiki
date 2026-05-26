import {
  postApiV1NodeSummary,
  putApiV1NodeDetail,
  V1NodeDetailResp,
} from '@/request';
import { useAppSelector } from '@/store';
import { message, Modal } from '@ctzhian/ui';
import { Button, CircularProgress, Stack, TextField } from '@mui/material';
import { useEffect, useState } from 'react';
import { useOutletContext } from 'react-router-dom';
import { WrapContext } from '..';
import { IconDJzhinengzhaiyao } from '@panda-wiki/icons';

interface SummaryProps {
  open: boolean;
  onClose: () => void;
  updateDetail: (detail: V1NodeDetailResp) => void;
}

type SummaryMode = 'chat' | 'analysis';

const Summary = ({ open, onClose, updateDetail }: SummaryProps) => {
  const { kb_id } = useAppSelector(state => state.config);
  const { nodeDetail } = useOutletContext<WrapContext>();
  const [summary, setSummary] = useState(nodeDetail?.meta?.summary || '');
  /** loadingMode 既是 loading 标志也是当前哪种生成中（互斥） */
  const [loadingMode, setLoadingMode] = useState<SummaryMode | null>(null);
  const [edit, setEdit] = useState(false);

  const handleClose = () => {
    setEdit(false);
    setSummary('');
    setLoadingMode(null);
    onClose();
  };

  const createSummary = (mode: SummaryMode) => {
    if (!nodeDetail || loadingMode) return;
    setLoadingMode(mode);
    postApiV1NodeSummary({ kb_id, ids: [nodeDetail.id!], mode })
      .then(res => {
        // @ts-expect-error 后端按 {summary} 返回
        setSummary(res.summary);
        setEdit(true);
      })
      .finally(() => {
        setLoadingMode(null);
      });
  };

  useEffect(() => {
    if (open) {
      setSummary(nodeDetail?.meta?.summary || '');
    }
  }, [open, nodeDetail]);

  const busy = loadingMode !== null;

  return (
    <Modal
      open={open}
      onCancel={handleClose}
      title='智能摘要'
      okText='保存'
      okButtonProps={{
        disabled: busy || !edit,
      }}
      onOk={() => {
        if (!nodeDetail) return;
        updateDetail({
          meta: {
            ...nodeDetail?.meta,
            summary,
          },
        });
        putApiV1NodeDetail({ id: nodeDetail.id!, kb_id, summary }).then(() => {
          message.success('保存成功');
        });
        handleClose();
      }}
    >
      <Stack gap={2}>
        <TextField
          autoFocus
          multiline
          disabled={busy}
          rows={10}
          fullWidth
          value={summary}
          onChange={e => {
            setSummary(e.target.value);
            setEdit(true);
          }}
          placeholder='请输入摘要'
        />
        <Stack direction='row' gap={1}>
          <Button
            fullWidth
            variant='contained'
            onClick={() => createSummary('chat')}
            disabled={busy}
            startIcon={
              loadingMode === 'chat' ? (
                <CircularProgress size={16} color='inherit' />
              ) : (
                <IconDJzhinengzhaiyao sx={{ fontSize: 16 }} />
              )
            }
          >
            {loadingMode === 'chat' ? '大模型生成中…' : '大模型摘要'}
          </Button>
          <Button
            fullWidth
            variant='outlined'
            onClick={() => createSummary('analysis')}
            disabled={busy}
            startIcon={
              loadingMode === 'analysis' ? (
                <CircularProgress size={16} />
              ) : (
                <IconDJzhinengzhaiyao sx={{ fontSize: 16 }} />
              )
            }
          >
            {loadingMode === 'analysis'
              ? '小模型生成中…'
              : '小模型摘要（非思考）'}
          </Button>
        </Stack>
      </Stack>
    </Modal>
  );
};

export default Summary;
