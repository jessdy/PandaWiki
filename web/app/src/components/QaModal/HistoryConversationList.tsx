import { ShareConversationHistoryItem } from '@/request/ShareConversation';
import AddIcon from '@mui/icons-material/Add';
import HistoryIcon from '@mui/icons-material/History';
import {
  Box,
  Button,
  CircularProgress,
  Stack,
  Typography,
  alpha,
} from '@mui/material';
import dayjs from 'dayjs';

interface HistoryConversationListProps {
  items: ShareConversationHistoryItem[];
  selectedId: string;
  loading: boolean;
  error: boolean;
  hasMore: boolean;
  onSelect: (item: ShareConversationHistoryItem) => void;
  onLoadMore: () => void;
  onNewConversation: () => void;
}

const HistoryConversationList = ({
  items,
  selectedId,
  loading,
  error,
  hasMore,
  onSelect,
  onLoadMore,
  onNewConversation,
}: HistoryConversationListProps) => (
  <Stack sx={{ height: '100%', minHeight: 0 }}>
    <Stack
      direction='row'
      alignItems='center'
      justifyContent='space-between'
      sx={{ px: 2, py: 1.75 }}
    >
      <Stack direction='row' alignItems='center' gap={0.75}>
        <HistoryIcon sx={{ fontSize: 18, color: 'primary.main' }} />
        <Typography sx={{ fontSize: 14, fontWeight: 600 }}>历史提问</Typography>
      </Stack>
    </Stack>

    <Box sx={{ px: 1.5, pb: 1.5 }}>
      <Button
        fullWidth
        size='small'
        variant='outlined'
        startIcon={<AddIcon />}
        onClick={onNewConversation}
      >
        新建提问
      </Button>
    </Box>

    <Stack
      gap={0.5}
      sx={{
        flex: 1,
        minHeight: 0,
        overflowY: 'auto',
        px: 1,
        pb: 1.5,
        scrollbarWidth: 'thin',
      }}
    >
      {items.map(item => {
        const selected = item.id === selectedId;
        return (
          <Button
            key={item.id}
            onClick={() => onSelect(item)}
            aria-current={selected ? 'page' : undefined}
            sx={theme => ({
              display: 'block',
              width: '100%',
              minHeight: 58,
              px: 1.25,
              py: 1,
              borderRadius: 1.5,
              textAlign: 'left',
              textTransform: 'none',
              color: 'text.primary',
              backgroundColor: selected
                ? alpha(theme.palette.primary.main, 0.1)
                : 'transparent',
              '&:hover': {
                backgroundColor: alpha(theme.palette.primary.main, 0.07),
              },
            })}
          >
            <Typography
              sx={{
                fontSize: 13,
                fontWeight: selected ? 600 : 500,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {item.subject || '未命名提问'}
            </Typography>
            <Typography color='text.secondary' sx={{ mt: 0.25, fontSize: 11 }}>
              {dayjs(item.created_at).format('MM-DD HH:mm')}
            </Typography>
          </Button>
        );
      })}

      {!loading && !error && items.length === 0 && (
        <Stack alignItems='center' gap={0.75} sx={{ px: 2, py: 5 }}>
          <HistoryIcon sx={{ fontSize: 26, color: 'text.disabled' }} />
          <Typography color='text.secondary' sx={{ fontSize: 12 }}>
            暂无历史提问
          </Typography>
        </Stack>
      )}

      {error && (
        <Stack alignItems='center' gap={1} sx={{ px: 2, py: 4 }}>
          <Typography color='error' sx={{ fontSize: 12 }}>
            历史记录加载失败
          </Typography>
          <Button size='small' onClick={onLoadMore}>
            重新加载
          </Button>
        </Stack>
      )}

      {loading && (
        <Stack alignItems='center' sx={{ py: 3 }}>
          <CircularProgress size={22} />
        </Stack>
      )}

      {!loading && !error && hasMore && (
        <Button size='small' onClick={onLoadMore} sx={{ mt: 0.5 }}>
          加载更多
        </Button>
      )}
    </Stack>
  </Stack>
);

export default HistoryConversationList;
