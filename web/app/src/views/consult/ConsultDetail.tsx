'use client';
import { useBasePath } from '@/hooks';
import { useStore } from '@/provider';
import {
  ConsultAttachment,
  ConsultInquiryDetail,
  ConsultStatus,
  getShareV1ConsultDetail,
  postShareV1ConsultClose,
  postShareV1ConsultReply,
} from '@/request/ShareConsult';
import { isSessionAuthInvalidError } from '@/utils/sessionAuthInvalid';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import HighlightOffOutlinedIcon from '@mui/icons-material/HighlightOffOutlined';
import {
  Box,
  Button,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
  Typography,
  alpha,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';
import AttachmentUploader from './AttachmentUploader';
import { notifyConsultRepliedCountChanged } from '@/components/header/ConsultNavButton';

dayjs.locale('zh-cn');

const STATUS_LABELS: Record<
  ConsultStatus,
  { label: string; color: 'default' | 'warning' | 'info' | 'success' | 'error' }
> = {
  pending: { label: '待处理', color: 'warning' },
  processing: { label: '处理中', color: 'info' },
  replied: { label: '已回复', color: 'success' },
  closed: { label: '已关闭', color: 'default' },
};

const renderAttachments = (atts: ConsultAttachment[]) => {
  if (!atts || atts.length === 0) return null;
  return (
    <Stack direction='row' gap={1} flexWrap='wrap' sx={{ mt: 1 }}>
      {atts.map((a, i) => (
        <Box key={`${a.url}-${i}`}>
          {a.type === 'image' ? (
            <Box
              component='a'
              href={a.url}
              target='_blank'
              rel='noopener noreferrer'
              sx={{
                display: 'inline-block',
                width: 96,
                height: 96,
                borderRadius: 1,
                overflow: 'hidden',
                border: '1px solid',
                borderColor: 'divider',
              }}
            >
              <Box
                component='img'
                src={a.url}
                alt={a.name || ''}
                sx={{ width: '100%', height: '100%', objectFit: 'cover' }}
              />
            </Box>
          ) : (
            <Box
              sx={{
                maxWidth: 360,
                borderRadius: 1,
                overflow: 'hidden',
                border: '1px solid',
                borderColor: 'divider',
                bgcolor: '#000',
              }}
            >
              <video
                src={a.url}
                controls
                style={{ display: 'block', width: '100%', maxHeight: 240 }}
              />
            </Box>
          )}
        </Box>
      ))}
    </Stack>
  );
};

interface ConsultDetailProps {
  id: number;
}

const ConsultDetail = ({ id }: ConsultDetailProps) => {
  const router = useRouter();
  const basePath = useBasePath();
  const { setLoginModalOpen, clearClientAuthInfo } = useStore();
  const [detail, setDetail] = useState<ConsultInquiryDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [reply, setReply] = useState('');
  const [replyAtts, setReplyAtts] = useState<ConsultAttachment[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [closing, setClosing] = useState(false);
  const [closeConfirmOpen, setCloseConfirmOpen] = useState(false);

  // 与 ConsultPage 同款：捕获后端 "请先登录" → 同步清前端缓存
  const handleAuthInvalid = useCallback(
    (e: unknown) => {
      if (!isSessionAuthInvalidError(e)) return false;
      clearClientAuthInfo?.();
      setLoginModalOpen?.(true);
      router.push(`${basePath}/consult`);
      return true;
    },
    [clearClientAuthInfo, setLoginModalOpen, router, basePath],
  );

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getShareV1ConsultDetail({ id });
      setDetail(res as unknown as ConsultInquiryDetail);
    } catch (e: unknown) {
      if (handleAuthInvalid(e)) return;
      message.error(
        (e as { message?: string })?.message || '加载失败，请稍后重试',
      );
    } finally {
      setLoading(false);
    }
  }, [id, handleAuthInvalid]);

  useEffect(() => {
    void load();
  }, [load]);

  const doReply = async () => {
    const text = reply.trim();
    if (!text) {
      message.error('请填写追问内容');
      return;
    }
    setSubmitting(true);
    try {
      await postShareV1ConsultReply({
        inquiry_id: id,
        content: text,
        attachments: replyAtts,
      });
      setReply('');
      setReplyAtts([]);
      await load();
      notifyConsultRepliedCountChanged();
      message.success('已追问');
    } catch (e: unknown) {
      if (handleAuthInvalid(e)) return;
      message.error((e as { message?: string })?.message || '提交失败');
    } finally {
      setSubmitting(false);
    }
  };

  const doClose = async () => {
    setClosing(true);
    try {
      await postShareV1ConsultClose({ inquiry_id: id });
      setCloseConfirmOpen(false);
      await load();
      notifyConsultRepliedCountChanged();
      message.success('已关闭该咨询');
    } catch (e: unknown) {
      if (handleAuthInvalid(e)) return;
      message.error((e as { message?: string })?.message || '关闭失败');
    } finally {
      setClosing(false);
    }
  };

  if (loading) {
    return (
      <Stack
        alignItems='center'
        justifyContent='center'
        sx={{ minHeight: '60vh' }}
      >
        <CircularProgress />
      </Stack>
    );
  }
  if (!detail) {
    return (
      <Stack alignItems='center' sx={{ py: 6 }}>
        <Typography color='text.secondary'>未找到该咨询</Typography>
        <Button
          sx={{ mt: 2 }}
          variant='outlined'
          onClick={() => router.push(`${basePath}/consult`)}
        >
          返回列表
        </Button>
      </Stack>
    );
  }

  const s = STATUS_LABELS[detail.status] || STATUS_LABELS.pending;
  const closed = detail.status === 'closed';

  return (
    <Stack
      sx={{
        py: 4,
        gap: 3,
      }}
    >
      <Stack direction='row' alignItems='center'>
        <Button
          startIcon={<ArrowBackIcon />}
          size='small'
          onClick={() => router.push(`${basePath}/consult`)}
        >
          返回
        </Button>
      </Stack>

      <Box>
        <Stack
          direction='row'
          alignItems='center'
          justifyContent='space-between'
          gap={2}
        >
          <Typography variant='h6' sx={{ fontWeight: 700 }}>
            {detail.title || detail.content.slice(0, 30)}
          </Typography>
          <Chip label={s.label} color={s.color} size='small' />
        </Stack>
        <Stack
          direction='row'
          gap={1.5}
          sx={{ mt: 0.5, fontSize: 12, color: 'text.secondary' }}
        >
          <Box component='span'>
            创建于 {dayjs(detail.created_at).format('YYYY-MM-DD HH:mm')}
          </Box>
          {detail.contact && (
            <Box component='span'>联系方式：{detail.contact}</Box>
          )}
        </Stack>
      </Box>

      {/* 消息流 */}
      <Stack gap={1.5}>
        {detail.messages.map(m => {
          const isUser = m.sender_kind === 'user';
          return (
            <Stack
              key={m.id}
              direction='row'
              justifyContent={isUser ? 'flex-end' : 'flex-start'}
            >
              <Box
                sx={{
                  maxWidth: '85%',
                  p: 1.5,
                  borderRadius: 2,
                  bgcolor: isUser ? 'primary.main' : 'background.paper',
                  color: isUser ? 'primary.contrastText' : 'text.primary',
                  border: isUser ? 'none' : '1px solid',
                  borderColor: 'divider',
                }}
              >
                <Stack
                  direction='row'
                  alignItems='center'
                  justifyContent='space-between'
                  gap={1}
                  sx={{ mb: 0.5 }}
                >
                  <Typography
                    variant='caption'
                    sx={{
                      fontWeight: 600,
                      opacity: isUser ? 0.9 : 0.7,
                    }}
                  >
                    {isUser
                      ? m.sender_name || '我'
                      : `${m.sender_name || '客服'} · 管理员`}
                  </Typography>
                  <Typography
                    variant='caption'
                    sx={{ opacity: isUser ? 0.85 : 0.55, fontSize: 11 }}
                  >
                    {dayjs(m.created_at).format('MM-DD HH:mm')}
                  </Typography>
                </Stack>
                <Typography
                  variant='body2'
                  sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}
                >
                  {m.content}
                </Typography>
                {renderAttachments(m.attachments)}
              </Box>
            </Stack>
          );
        })}
      </Stack>

      {/* 关闭操作 */}
      {!closed && (
        <Box
          sx={{
            p: 2,
            borderRadius: 2,
            border: '1px solid',
            borderColor: 'error.main',
            bgcolor: theme => alpha(theme.palette.error.main, 0.08),
            display: 'flex',
            alignItems: { xs: 'stretch', sm: 'center' },
            justifyContent: 'space-between',
            flexDirection: { xs: 'column', sm: 'row' },
            gap: 1.5,
          }}
        >
          <Typography
            variant='body2'
            sx={{ color: 'error.main', lineHeight: 1.6, fontWeight: 500 }}
          >
            问题已解决？点击关闭后将无法继续追问。
          </Typography>
          <Button
            variant='contained'
            color='error'
            size='medium'
            startIcon={<HighlightOffOutlinedIcon />}
            disabled={closing || submitting}
            onClick={() => setCloseConfirmOpen(true)}
            sx={{
              flexShrink: 0,
              fontWeight: 600,
              px: 2.5,
              boxShadow: '0 4px 12px rgba(211, 47, 47, 0.35)',
            }}
          >
            {closing ? '关闭中…' : '关闭问题'}
          </Button>
        </Box>
      )}

      <Dialog
        open={closeConfirmOpen}
        onClose={() => {
          if (!closing) setCloseConfirmOpen(false);
        }}
        maxWidth='xs'
        fullWidth
        PaperProps={{ sx: { borderRadius: 2.5 } }}
      >
        <DialogTitle sx={{ pb: 1, fontWeight: 700, fontSize: 18 }}>
          确认关闭问题
        </DialogTitle>
        <DialogContent>
          <Stack direction='row' gap={1.5} alignItems='flex-start'>
            <Box
              sx={{
                width: 40,
                height: 40,
                borderRadius: '50%',
                bgcolor: theme => alpha(theme.palette.error.main, 0.12),
                color: 'error.main',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                flexShrink: 0,
              }}
            >
              <HighlightOffOutlinedIcon fontSize='small' />
            </Box>
            <Stack gap={0.75}>
              <Typography variant='body2' sx={{ fontWeight: 600 }}>
                关闭后将无法继续追问
              </Typography>
              <Typography
                variant='body2'
                color='text.secondary'
                sx={{ lineHeight: 1.6 }}
              >
                若问题尚未解决，建议先继续追问或等待管理员回复。确认要关闭这条咨询吗？
              </Typography>
            </Stack>
          </Stack>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5, pt: 0, gap: 1 }}>
          <Button
            variant='outlined'
            color='inherit'
            disabled={closing}
            onClick={() => setCloseConfirmOpen(false)}
            sx={{ minWidth: 88 }}
          >
            取消
          </Button>
          <Button
            variant='contained'
            color='error'
            disabled={closing}
            onClick={() => void doClose()}
            sx={{ minWidth: 108, fontWeight: 600 }}
          >
            {closing ? '关闭中…' : '确认关闭'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* 追问表单 */}
      {closed ? (
        <Box
          sx={{
            p: 2,
            borderRadius: 2,
            border: '1px dashed',
            borderColor: 'divider',
            textAlign: 'center',
            color: 'text.secondary',
            fontSize: 13,
          }}
        >
          该咨询已关闭，不再支持追问。
        </Box>
      ) : (
        <Box
          sx={{
            p: 2,
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 2,
            bgcolor: 'background.paper',
          }}
        >
          <Typography variant='subtitle2' sx={{ fontWeight: 600, mb: 1.5 }}>
            继续追问
          </Typography>
          <Stack gap={1.5}>
            <TextField
              fullWidth
              multiline
              minRows={3}
              maxRows={10}
              placeholder='补充说明、提供更多信息…'
              value={reply}
              onChange={e => setReply(e.target.value)}
              inputProps={{ maxLength: 8000 }}
              disabled={submitting}
            />
            <AttachmentUploader
              value={replyAtts}
              onChange={setReplyAtts}
              disabled={submitting}
            />
            <Stack direction='row' justifyContent='flex-end'>
              <Button
                variant='contained'
                onClick={() => void doReply()}
                disabled={submitting || !reply.trim()}
              >
                {submitting ? '提交中…' : '发送'}
              </Button>
            </Stack>
          </Stack>
        </Box>
      )}
    </Stack>
  );
};

export default ConsultDetail;
