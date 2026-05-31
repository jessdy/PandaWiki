'use client';
import { useBasePath } from '@/hooks';
import { useStore } from '@/provider';
import {
  ConsultAttachment,
  ConsultInquiryListItem,
  ConsultStatus,
  getShareV1ConsultList,
  postShareV1ConsultCreate,
} from '@/request/ShareConsult';
import { isAuthInfoEmpty } from '@/utils/authInfo';
import { isSessionAuthInvalidError } from '@/utils/sessionAuthInvalid';
import { lacksAccountIdentityForSiteFeedback } from '@/utils/siteFeedbackAuth';
import {
  Box,
  Button,
  Chip,
  CircularProgress,
  Divider,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';
import AttachmentUploader from './AttachmentUploader';
import { notifyConsultRepliedCountChanged } from '@/components/header/ConsultNavButton';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

/* ---------------------------------------------------------------- */
/* 状态展示                                                          */
/* ---------------------------------------------------------------- */

const STATUS_LABELS: Record<
  ConsultStatus,
  { label: string; color: 'default' | 'warning' | 'info' | 'success' | 'error' }
> = {
  pending: { label: '待处理', color: 'warning' },
  processing: { label: '处理中', color: 'info' },
  replied: { label: '已回复', color: 'success' },
  closed: { label: '已关闭', color: 'default' },
};

/* ---------------------------------------------------------------- */
/* 页面主体                                                          */
/* ---------------------------------------------------------------- */

const ConsultPage = () => {
  const router = useRouter();
  const basePath = useBasePath();
  const { authInfo, setLoginModalOpen, clearClientAuthInfo } = useStore();

  /**
   * 处理「后端 session 失效」错误：
   * 前端 localStorage authInfo 可能因为长期未刷新而过期，但 cookie/session 已失效；
   * 此时调任意鉴权接口会回 "请先登录"。一旦命中：清掉本地 authInfo + 打开登录框，
   * 状态归一，用户重新登录后所有页面状态一致。
   */
  const handleAuthInvalid = useCallback(
    (e: unknown) => {
      if (!isSessionAuthInvalidError(e)) return false;
      clearClientAuthInfo?.();
      setLoginModalOpen?.(true);
      return true;
    },
    [clearClientAuthInfo, setLoginModalOpen],
  );

  // 表单状态
  const [content, setContent] = useState('');
  const [contact, setContact] = useState('');
  const [attachments, setAttachments] = useState<ConsultAttachment[]>([]);
  const [submitting, setSubmitting] = useState(false);

  // 列表状态
  const [list, setList] = useState<ConsultInquiryListItem[]>([]);
  const [listLoading, setListLoading] = useState(false);

  // SSR/hydration 守卫：避免首帧把"未登录"占位渲染出来引起闪烁。
  // useStore() 的 authInfo 是从 localStorage 同步初始化的（lazy init），但 SSR 阶段拿不到 window，
  // 必须等 mount 后再读 authInfo 才有意义。
  const [hydrated, setHydrated] = useState(false);
  useEffect(() => {
    setHydrated(true);
  }, []);

  const needsLogin = useCallback(() => {
    if (isAuthInfoEmpty(authInfo)) {
      message.warning('请先登录');
      setLoginModalOpen?.(true);
      return true;
    }
    if (lacksAccountIdentityForSiteFeedback(authInfo)) {
      message.error(
        '疑难咨询需使用账号登录（用户名密码或企业 SSO），当前为访问口令认证',
      );
      setLoginModalOpen?.(true);
      return true;
    }
    return false;
  }, [authInfo, setLoginModalOpen]);

  const loadList = useCallback(async () => {
    if (isAuthInfoEmpty(authInfo)) return;
    setListLoading(true);
    try {
      const res = await getShareV1ConsultList({ page: 1, per_page: 30 });
      // httpClient 已 unwrap data；返回的是 { items, total }
      const data = res as { items?: ConsultInquiryListItem[] };
      setList(data?.items || []);
      notifyConsultRepliedCountChanged();
    } catch (e) {
      // 命中"session 已失效"时清掉本地缓存并弹登录框（前端 localStorage
      // 可能比后端 session 活得更久，状态不一致需主动归零）
      handleAuthInvalid(e);
    } finally {
      setListLoading(false);
    }
  }, [authInfo, handleAuthInvalid]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  const submit = async () => {
    if (needsLogin()) return;
    const text = content.trim();
    if (!text) {
      message.error('请填写咨询内容');
      return;
    }
    setSubmitting(true);
    try {
      await postShareV1ConsultCreate({
        content: text,
        contact: contact.trim(),
        attachments,
      });
      message.success('已提交，我们会尽快处理');
      setContent('');
      setContact('');
      setAttachments([]);
      void loadList();
    } catch (e: unknown) {
      if (handleAuthInvalid(e)) return;
      message.error(
        (e as { message?: string })?.message || '提交失败，请稍后重试',
      );
    } finally {
      setSubmitting(false);
    }
  };

  const openDetail = (id: number) => {
    router.push(`${basePath}/consult/${id}`);
  };

  return (
    <Stack
      sx={{
        py: 4,
        gap: 4,
      }}
    >
      <Box>
        <Typography variant='h5' sx={{ fontWeight: 700, mb: 0.5 }}>
          疑难咨询
        </Typography>
        <Typography variant='body2' color='text.secondary'>
          有问题随时提交，我们会在后台收到并尽快回复。下方可查看您过往的咨询和回复进度。
        </Typography>
      </Box>

      {/* 上半：提问表单 */}
      <Box
        sx={{
          p: { xs: 2, sm: 3 },
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 2,
          bgcolor: 'background.paper',
        }}
      >
        <Typography variant='subtitle1' sx={{ fontWeight: 600, mb: 2 }}>
          我要提问
        </Typography>
        <Stack gap={2}>
          <TextField
            fullWidth
            multiline
            minRows={4}
            maxRows={12}
            placeholder='请详细描述您遇到的问题或需要帮助的场景…'
            value={content}
            onChange={e => setContent(e.target.value)}
            inputProps={{ maxLength: 8000 }}
            disabled={submitting}
          />
          <TextField
            fullWidth
            size='small'
            placeholder='联系方式（方便我们回复时联系到您）'
            value={contact}
            onChange={e => setContact(e.target.value)}
            inputProps={{ maxLength: 200 }}
            disabled={submitting}
          />
          <AttachmentUploader
            value={attachments}
            onChange={setAttachments}
            disabled={submitting}
          />
          <Stack direction='row' justifyContent='flex-end'>
            <Button
              variant='contained'
              onClick={() => void submit()}
              disabled={submitting || !content.trim()}
            >
              {submitting ? '提交中…' : '提交咨询'}
            </Button>
          </Stack>
        </Stack>
      </Box>

      {/* 下半：我的咨询列表 */}
      <Box>
        <Stack
          direction='row'
          alignItems='center'
          justifyContent='space-between'
          sx={{ mb: 1.5 }}
        >
          <Typography variant='subtitle1' sx={{ fontWeight: 600 }}>
            我的咨询
            {list.length > 0 && (
              <Typography
                component='span'
                variant='body2'
                color='text.secondary'
                sx={{ ml: 1, fontWeight: 400 }}
              >
                共 {list.length} 条
              </Typography>
            )}
          </Typography>
        </Stack>
        <Box
          sx={{
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 2,
            bgcolor: 'background.paper',
          }}
        >
          {!hydrated ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
              <CircularProgress size={22} />
            </Box>
          ) : isAuthInfoEmpty(authInfo) ? (
            <Box sx={{ p: 3, textAlign: 'center' }}>
              <Typography variant='body2' color='text.secondary' sx={{ mb: 1 }}>
                登录后才能查看历史咨询
              </Typography>
              <Button
                size='small'
                variant='outlined'
                onClick={() => setLoginModalOpen?.(true)}
              >
                去登录
              </Button>
            </Box>
          ) : listLoading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
              <CircularProgress size={22} />
            </Box>
          ) : list.length === 0 ? (
            <Box sx={{ p: 3 }}>
              <Typography variant='body2' color='text.secondary'>
                还没有咨询记录。提交后会显示在这里。
              </Typography>
            </Box>
          ) : (
            list.map((item, idx) => {
              const s = STATUS_LABELS[item.status] || STATUS_LABELS.pending;
              return (
                <Box key={item.id}>
                  {idx > 0 && <Divider />}
                  <Box
                    sx={{
                      p: 2,
                      cursor: 'pointer',
                      transition: 'background-color 0.15s',
                      '&:hover': { bgcolor: 'action.hover' },
                    }}
                    onClick={() => openDetail(item.id)}
                  >
                    <Stack
                      direction='row'
                      alignItems='center'
                      justifyContent='space-between'
                      gap={2}
                    >
                      <Stack sx={{ flex: 1, minWidth: 0 }} gap={0.5}>
                        <Typography
                          variant='body2'
                          sx={{
                            fontWeight: 600,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            display: '-webkit-box',
                            WebkitLineClamp: 1,
                            WebkitBoxOrient: 'vertical',
                          }}
                          title={item.title || item.content}
                        >
                          {item.title || item.content.slice(0, 60)}
                        </Typography>
                        <Stack
                          direction='row'
                          gap={1.5}
                          alignItems='center'
                          sx={{ fontSize: 12, color: 'text.secondary' }}
                        >
                          <Box component='span'>
                            创建：
                            {dayjs(item.created_at).format('MM-DD HH:mm')}
                          </Box>
                          {item.reply_count > 0 && (
                            <Box component='span'>
                              {item.reply_count} 条后续
                            </Box>
                          )}
                          <Box component='span'>
                            最近：
                            {dayjs(item.last_message_at).fromNow()}
                          </Box>
                        </Stack>
                      </Stack>
                      <Chip
                        label={s.label}
                        color={s.color}
                        size='small'
                        sx={{ flexShrink: 0 }}
                      />
                    </Stack>
                  </Box>
                </Box>
              );
            })
          )}
        </Box>
      </Box>
    </Stack>
  );
};

export default ConsultPage;
