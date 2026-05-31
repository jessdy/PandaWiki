import {
  ConsultAttachment,
  ConsultInquiryDetail,
  ConsultStatus,
  getApiV1ConsultDetail,
  postApiV1ConsultReply,
  postApiV1ConsultStatus,
} from '@/request/Consult';
import {
  Box,
  Button,
  Checkbox,
  Chip,
  CircularProgress,
  Drawer,
  Divider,
  FormControl,
  FormControlLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import dayjs from 'dayjs';
import { useCallback, useEffect, useState } from 'react';
import AdminAttachmentUploader from './AdminAttachmentUploader';

const STATUS_LABELS: Record<
  ConsultStatus,
  { label: string; color: 'default' | 'warning' | 'info' | 'success' }
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
                maxWidth: 320,
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
                style={{ display: 'block', width: '100%', maxHeight: 220 }}
              />
            </Box>
          )}
        </Box>
      ))}
    </Stack>
  );
};

export interface ConsultDetailDrawerProps {
  id: number | null;
  onClose: () => void;
  /** 关闭抽屉时若有变化（回复 / 改状态）通知外层刷列表 */
  onChanged: () => void;
}

const ConsultDetailDrawer = ({
  id,
  onClose,
  onChanged,
}: ConsultDetailDrawerProps) => {
  const open = id !== null;
  const [detail, setDetail] = useState<ConsultInquiryDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [reply, setReply] = useState('');
  const [replyAtts, setReplyAtts] = useState<ConsultAttachment[]>([]);
  const [markReplied, setMarkReplied] = useState(true);
  const [submitting, setSubmitting] = useState(false);

  // 关键：dirty 用于判断关闭抽屉时是否要触发列表 reload，避免无谓刷新。
  const [dirty, setDirty] = useState(false);

  const load = useCallback(async () => {
    if (id == null) return;
    setLoading(true);
    try {
      const res = await getApiV1ConsultDetail({ id });
      setDetail(res as unknown as ConsultInquiryDetail);
    } catch (e: unknown) {
      message.error((e as { message?: string })?.message || '加载失败');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (open) {
      setReply('');
      setReplyAtts([]);
      setMarkReplied(true);
      setDirty(false);
      void load();
    } else {
      setDetail(null);
    }
  }, [open, load]);

  const handleClose = () => {
    if (dirty) onChanged();
    onClose();
  };

  const doReply = async () => {
    if (id == null) return;
    const text = reply.trim();
    if (!text) {
      message.error('回复内容不能为空');
      return;
    }
    setSubmitting(true);
    try {
      await postApiV1ConsultReply({
        inquiry_id: id,
        content: text,
        attachments: replyAtts,
        mark_replied: markReplied,
      });
      setReply('');
      setReplyAtts([]);
      setDirty(true);
      message.success('已回复');
      await load();
      window.dispatchEvent(new Event('consult-open-count-changed'));
    } catch (e: unknown) {
      message.error((e as { message?: string })?.message || '回复失败');
    } finally {
      setSubmitting(false);
    }
  };

  const doSetStatus = async (status: ConsultStatus) => {
    if (id == null || !detail) return;
    if (status === detail.status) return;
    try {
      await postApiV1ConsultStatus({ inquiry_id: id, status });
      setDirty(true);
      message.success('状态已更新');
      await load();
      window.dispatchEvent(new Event('consult-open-count-changed'));
    } catch (e: unknown) {
      message.error((e as { message?: string })?.message || '更新失败');
    }
  };

  return (
    <Drawer
      anchor='right'
      open={open}
      onClose={handleClose}
      PaperProps={{ sx: { width: { xs: '100%', sm: 720 } } }}
    >
      <Stack sx={{ height: '100%' }}>
        {/* 头部 */}
        <Box sx={{ p: 2, borderBottom: '1px solid', borderColor: 'divider' }}>
          {loading || !detail ? (
            <Stack
              direction='row'
              alignItems='center'
              gap={1}
              sx={{ minHeight: 28 }}
            >
              <CircularProgress size={14} />
              <Typography variant='body2' color='text.secondary'>
                加载中…
              </Typography>
            </Stack>
          ) : (
            <Stack gap={1}>
              <Stack
                direction='row'
                alignItems='center'
                justifyContent='space-between'
                gap={1}
              >
                <Typography variant='subtitle1' sx={{ fontWeight: 600 }}>
                  {detail.title || detail.content.slice(0, 30) || '咨询详情'}
                </Typography>
                <FormControl size='small' sx={{ minWidth: 120 }}>
                  <Select
                    value={detail.status}
                    onChange={e =>
                      void doSetStatus(e.target.value as ConsultStatus)
                    }
                    renderValue={v => {
                      const s = STATUS_LABELS[v as ConsultStatus];
                      return (
                        <Chip
                          label={s?.label || v}
                          color={s?.color || 'default'}
                          size='small'
                        />
                      );
                    }}
                  >
                    {(Object.keys(STATUS_LABELS) as ConsultStatus[]).map(s => (
                      <MenuItem key={s} value={s}>
                        {STATUS_LABELS[s].label}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Stack>
              <Stack
                direction='row'
                gap={1.5}
                flexWrap='wrap'
                sx={{ fontSize: 12, color: 'text.secondary' }}
              >
                <Box component='span'>
                  创建于 {dayjs(detail.created_at).format('YYYY-MM-DD HH:mm')}
                </Box>
                <Box component='span'>用户 ID：{detail.user_id || '-'}</Box>
                {detail.contact && (
                  <Box component='span'>联系方式：{detail.contact}</Box>
                )}
              </Stack>
            </Stack>
          )}
        </Box>

        {/* 消息流 */}
        <Box sx={{ flex: 1, overflowY: 'auto', p: 2, bgcolor: 'grey.50' }}>
          {loading || !detail ? (
            <Stack alignItems='center' sx={{ py: 4 }}>
              <CircularProgress size={20} />
            </Stack>
          ) : detail.messages.length === 0 ? (
            <Typography variant='body2' color='text.secondary'>
              暂无消息
            </Typography>
          ) : (
            <Stack gap={1.5}>
              {detail.messages.map(m => {
                const isAdmin = m.sender_kind === 'admin';
                return (
                  <Stack
                    key={m.id}
                    direction='row'
                    justifyContent={isAdmin ? 'flex-end' : 'flex-start'}
                  >
                    <Box
                      sx={{
                        maxWidth: '85%',
                        p: 1.25,
                        borderRadius: 2,
                        bgcolor: isAdmin ? 'primary.main' : 'background.paper',
                        color: isAdmin
                          ? 'primary.contrastText'
                          : 'text.primary',
                        border: isAdmin ? 'none' : '1px solid',
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
                            opacity: isAdmin ? 0.9 : 0.7,
                          }}
                        >
                          {isAdmin
                            ? `${m.sender_name || '管理员'} · 管理员`
                            : m.sender_name || '提问人'}
                        </Typography>
                        <Typography
                          variant='caption'
                          sx={{ opacity: isAdmin ? 0.85 : 0.55, fontSize: 11 }}
                        >
                          {dayjs(m.created_at).format('MM-DD HH:mm')}
                        </Typography>
                      </Stack>
                      <Typography
                        variant='body2'
                        sx={{
                          fontSize: 13,
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-word',
                        }}
                      >
                        {m.content}
                      </Typography>
                      {renderAttachments(m.attachments)}
                    </Box>
                  </Stack>
                );
              })}
            </Stack>
          )}
        </Box>

        <Divider />

        {/* 回复表单 */}
        {detail?.status === 'closed' ? (
          <Box
            sx={{
              p: 2,
              textAlign: 'center',
              color: 'text.secondary',
              fontSize: 13,
            }}
          >
            该咨询已关闭，如需继续沟通请先切回「处理中」。
          </Box>
        ) : (
          <Box sx={{ p: 2 }}>
            <Typography variant='subtitle2' sx={{ fontWeight: 600, mb: 1 }}>
              管理员回复
            </Typography>
            <Stack gap={1.5}>
              <TextField
                fullWidth
                multiline
                minRows={3}
                maxRows={10}
                placeholder='回复内容…'
                value={reply}
                onChange={e => setReply(e.target.value)}
                inputProps={{ maxLength: 8000 }}
                disabled={submitting}
              />
              <AdminAttachmentUploader
                value={replyAtts}
                onChange={setReplyAtts}
                disabled={submitting}
              />
              <Stack
                direction='row'
                alignItems='center'
                justifyContent='space-between'
                gap={1}
              >
                <FormControlLabel
                  control={
                    <Checkbox
                      size='small'
                      checked={markReplied}
                      onChange={e => setMarkReplied(e.target.checked)}
                    />
                  }
                  label={
                    <Typography variant='caption' color='text.secondary'>
                      回复后置为「已回复」
                    </Typography>
                  }
                />
                <Button
                  variant='contained'
                  onClick={() => void doReply()}
                  disabled={submitting || !reply.trim()}
                >
                  {submitting ? '提交中…' : '发送回复'}
                </Button>
              </Stack>
            </Stack>
          </Box>
        )}
      </Stack>
    </Drawer>
  );
};

export default ConsultDetailDrawer;
