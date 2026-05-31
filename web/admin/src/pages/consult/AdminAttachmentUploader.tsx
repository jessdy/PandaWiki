import { uploadFile } from '@/api';
import { ConsultAttachment } from '@/request/Consult';
import CloseIcon from '@mui/icons-material/Close';
import ImageIcon from '@mui/icons-material/Image';
import VideoLibraryIcon from '@mui/icons-material/VideoLibrary';
import {
  Box,
  Button,
  CircularProgress,
  IconButton,
  Stack,
  Typography,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import { useRef, useState } from 'react';

// 与前台 ConsultAttachmentUploader 保持一致的限制（宽松档）
const LIMITS = {
  image: { maxSize: 20 * 1024 * 1024, maxCount: 10 },
  video: { maxSize: 200 * 1024 * 1024, maxCount: 3 },
};

export interface AdminAttachmentUploaderProps {
  value: ConsultAttachment[];
  onChange: (next: ConsultAttachment[]) => void;
  disabled?: boolean;
}

const AdminAttachmentUploader = ({
  value,
  onChange,
  disabled,
}: AdminAttachmentUploaderProps) => {
  const imgInputRef = useRef<HTMLInputElement>(null);
  const vidInputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);

  const counts = {
    image: value.filter(v => v.type === 'image').length,
    video: value.filter(v => v.type === 'video').length,
  };

  const handlePick = async (
    type: 'image' | 'video',
    files: FileList | null,
  ) => {
    if (!files || files.length === 0) return;
    const remaining = LIMITS[type].maxCount - counts[type];
    if (remaining <= 0) {
      message.warning(
        `${type === 'image' ? '图片' : '视频'}已达上限 ${LIMITS[type].maxCount} 个`,
      );
      return;
    }
    const toUpload: File[] = [];
    for (const f of Array.from(files).slice(0, remaining)) {
      if (type === 'image' && !f.type.startsWith('image/')) {
        message.error(`${f.name}：仅支持图片格式`);
        continue;
      }
      if (type === 'video' && !f.type.startsWith('video/')) {
        message.error(`${f.name}：仅支持视频格式`);
        continue;
      }
      if (f.size > LIMITS[type].maxSize) {
        const mb = (LIMITS[type].maxSize / 1024 / 1024) | 0;
        message.error(`${f.name}：单文件不能超过 ${mb}MB`);
        continue;
      }
      toUpload.push(f);
    }
    if (toUpload.length === 0) return;

    setUploading(true);
    try {
      const next = [...value];
      for (const f of toUpload) {
        const form = new FormData();
        form.append('file', f);
        const res = await uploadFile(form);
        if (!res?.key) {
          message.error(`${f.name}：上传失败`);
          continue;
        }
        next.push({
          type,
          url: '/static-file/' + res.key,
          name: f.name,
          size: f.size,
          mime: f.type,
        });
      }
      onChange(next);
    } catch (e: unknown) {
      message.error((e as { message?: string })?.message || '上传失败');
    } finally {
      setUploading(false);
    }
  };

  const remove = (idx: number) => {
    const next = [...value];
    next.splice(idx, 1);
    onChange(next);
  };

  return (
    <Stack gap={1}>
      <Stack direction='row' gap={1} alignItems='center' flexWrap='wrap'>
        <input
          ref={imgInputRef}
          type='file'
          hidden
          multiple
          accept='image/*'
          onChange={e => {
            void handlePick('image', e.target.files);
            if (imgInputRef.current) imgInputRef.current.value = '';
          }}
        />
        <input
          ref={vidInputRef}
          type='file'
          hidden
          multiple
          accept='video/*'
          onChange={e => {
            void handlePick('video', e.target.files);
            if (vidInputRef.current) vidInputRef.current.value = '';
          }}
        />
        <Button
          size='small'
          variant='outlined'
          startIcon={<ImageIcon />}
          disabled={
            disabled || uploading || counts.image >= LIMITS.image.maxCount
          }
          onClick={() => imgInputRef.current?.click()}
        >
          图片{' '}
          {counts.image > 0 ? `(${counts.image}/${LIMITS.image.maxCount})` : ''}
        </Button>
        <Button
          size='small'
          variant='outlined'
          startIcon={<VideoLibraryIcon />}
          disabled={
            disabled || uploading || counts.video >= LIMITS.video.maxCount
          }
          onClick={() => vidInputRef.current?.click()}
        >
          视频{' '}
          {counts.video > 0 ? `(${counts.video}/${LIMITS.video.maxCount})` : ''}
        </Button>
        {uploading && (
          <Stack direction='row' alignItems='center' gap={0.5}>
            <CircularProgress size={14} />
            <Typography variant='caption' color='text.secondary'>
              上传中…
            </Typography>
          </Stack>
        )}
      </Stack>
      {value.length > 0 && (
        <Stack direction='row' gap={1} flexWrap='wrap'>
          {value.map((a, idx) => (
            <Box
              key={`${a.url}-${idx}`}
              sx={{
                position: 'relative',
                width: 80,
                height: 80,
                border: '1px solid',
                borderColor: 'divider',
                borderRadius: 1,
                overflow: 'hidden',
                bgcolor: 'background.paper',
              }}
            >
              {a.type === 'image' ? (
                <Box
                  component='img'
                  src={a.url}
                  alt={a.name || ''}
                  sx={{ width: '100%', height: '100%', objectFit: 'cover' }}
                />
              ) : (
                <Stack
                  alignItems='center'
                  justifyContent='center'
                  sx={{ width: '100%', height: '100%', fontSize: 11, p: 0.5 }}
                >
                  <VideoLibraryIcon />
                  <Typography
                    variant='caption'
                    noWrap
                    sx={{ width: '100%', textAlign: 'center' }}
                    title={a.name}
                  >
                    {a.name || '视频'}
                  </Typography>
                </Stack>
              )}
              <IconButton
                size='small'
                disabled={disabled}
                onClick={() => remove(idx)}
                sx={{
                  position: 'absolute',
                  right: 2,
                  top: 2,
                  width: 18,
                  height: 18,
                  bgcolor: 'rgba(0,0,0,0.45)',
                  color: '#fff',
                  '&:hover': { bgcolor: 'rgba(0,0,0,0.65)' },
                }}
              >
                <CloseIcon sx={{ fontSize: 12 }} />
              </IconButton>
            </Box>
          ))}
        </Stack>
      )}
    </Stack>
  );
};

export default AdminAttachmentUploader;
