'use client';

import {
  getShareV1MethodRulesCategories,
  postShareV1MethodRulesMatch,
  WorkModeCategory,
} from '@/request/ShareMethodRule';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  FormControl,
  MenuItem,
  Select,
  Stack,
  Typography,
} from '@mui/material';
import { useCallback, useEffect, useState } from 'react';
import AttributePanel, { AttributePanelProps } from './AttributePanel';
import { AttributePanelMeta } from '@/utils/attributePanelParse';

interface ManualWorkModeSelectorProps {
  workChrome: AttributePanelProps['workChrome'];
}

const ManualWorkModeSelector = ({
  workChrome,
}: ManualWorkModeSelectorProps) => {
  const [categories, setCategories] = useState<WorkModeCategory[]>([]);
  const [selectedCategory, setSelectedCategory] = useState('');
  const [panelMeta, setPanelMeta] = useState<AttributePanelMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [matching, setMatching] = useState(false);
  const [error, setError] = useState('');

  const loadCategories = useCallback(() => {
    setLoading(true);
    setError('');
    getShareV1MethodRulesCategories()
      .then(response => {
        setCategories(response?.items || []);
      })
      .catch(() => setError('品类加载失败，请稍后重试。'))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    loadCategories();
  }, [loadCategories]);

  const handleCategoryChange = (categoryName: string) => {
    setSelectedCategory(categoryName);
    setPanelMeta(null);
    setError('');
    if (!categoryName) return;

    const category = categories.find(item => item.name === categoryName);
    setMatching(true);
    postShareV1MethodRulesMatch({
      category: categoryName,
      collected: {},
    })
      .then(response => {
        setPanelMeta({
          category: categoryName,
          specs: response?.specs || category?.specs || [],
          collected: {},
          methods: response?.methods || [],
        });
      })
      .catch(() => {
        setPanelMeta({
          category: categoryName,
          specs: category?.specs || [],
          collected: {},
          methods: [],
        });
        setError('解决方案匹配失败，可重新选择品类后再试。');
      })
      .finally(() => setMatching(false));
  };

  if (loading) {
    return (
      <Stack
        alignItems='center'
        justifyContent='center'
        gap={1.5}
        sx={{ flex: 1 }}
      >
        <CircularProgress size={26} sx={{ color: workChrome?.gold }} />
        <Typography sx={{ fontSize: 13, color: workChrome?.textMuted }}>
          正在加载品类…
        </Typography>
      </Stack>
    );
  }

  if (error && categories.length === 0) {
    return (
      <Stack
        alignItems='center'
        justifyContent='center'
        gap={1.5}
        sx={{ flex: 1 }}
      >
        <Alert severity='error'>{error}</Alert>
        <Button variant='outlined' onClick={loadCategories}>
          重新加载
        </Button>
      </Stack>
    );
  }

  if (categories.length === 0) {
    return (
      <Stack
        alignItems='center'
        justifyContent='center'
        gap={1}
        sx={{ flex: 1 }}
      >
        <Typography sx={{ fontSize: 14, color: workChrome?.textPrimary }}>
          暂无可选品类
        </Typography>
        <Typography sx={{ fontSize: 12, color: workChrome?.textMuted }}>
          请先在后台为当前知识库配置实战品类和属性。
        </Typography>
      </Stack>
    );
  }

  return (
    <Stack gap={2} sx={{ flex: 1, minHeight: 0, overflowY: 'auto', pb: 1 }}>
      <Box
        sx={{
          p: 2,
          borderRadius: '10px',
          border: `1px solid ${workChrome?.borderStrong}`,
          backgroundColor: workChrome?.bgRaised,
        }}
      >
        <Typography
          sx={{
            mb: 1,
            fontSize: 14,
            fontWeight: 600,
            color: workChrome?.textPrimary,
          }}
        >
          直接选择品类和属性
        </Typography>
        <Typography
          sx={{ mb: 1.5, fontSize: 12, color: workChrome?.textMuted }}
        >
          无需上传图片，选择品类后补全属性即可匹配解决方案。
        </Typography>
        <FormControl fullWidth size='small'>
          <Select
            value={selectedCategory}
            displayEmpty
            onChange={event => handleCategoryChange(event.target.value)}
            disabled={matching}
            aria-label='选择品类'
            sx={{
              color: workChrome?.textPrimary,
              backgroundColor: workChrome?.bgRaised,
              '& .MuiOutlinedInput-notchedOutline': {
                borderColor: workChrome?.border,
              },
              '&:hover .MuiOutlinedInput-notchedOutline': {
                borderColor: workChrome?.borderStrong,
              },
            }}
          >
            <MenuItem value=''>
              <em>请选择品类</em>
            </MenuItem>
            {categories.map(category => (
              <MenuItem key={category.name} value={category.name}>
                {category.name}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>

      {matching && (
        <Stack
          direction='row'
          alignItems='center'
          justifyContent='center'
          gap={1}
        >
          <CircularProgress size={18} sx={{ color: workChrome?.gold }} />
          <Typography sx={{ fontSize: 12, color: workChrome?.textMuted }}>
            正在读取属性和解决方案…
          </Typography>
        </Stack>
      )}

      {error && categories.length > 0 && (
        <Alert severity='warning'>{error}</Alert>
      )}

      {!matching && panelMeta && (
        <AttributePanel
          key={panelMeta.category}
          meta={panelMeta}
          workChrome={workChrome}
          source='manual'
        />
      )}
    </Stack>
  );
};

export default ManualWorkModeSelector;
