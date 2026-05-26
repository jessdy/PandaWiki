import {
  CategoryPromptItem,
  getApiV1CategoryPrompts,
} from '@/request/CategoryPrompt';
import {
  ImageDescriptionTemplate,
  getApiV1ImageDescriptionTemplates,
  postApiV1ImageDescriptionTemplates,
} from '@/request/ImageDescriptionTemplate';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItemButton,
  ListItemText,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import { useCallback, useEffect, useMemo, useState } from 'react';

/* ------------------------------------------------------------------ */
/* Helpers                                                             */
/* ------------------------------------------------------------------ */

/** 解析品类的「属性维护」字符串（支持中英文逗号）。 */
function splitAttrKeys(raw?: string): string[] {
  if (!raw) return [];
  return raw
    .replace(/\uff0c/g, ',')
    .split(',')
    .map(s => s.trim())
    .filter(Boolean);
}

/**
 * 把模版的属性键值对渲染成图片 title 文本：「品牌：Apple，颜色：红色」。
 * 用品类配置中的属性顺序，避免 map 迭代顺序不稳定；缺失/空值的属性会被跳过。
 */
function renderAsDescription(
  attrKeys: string[],
  values: Record<string, string>,
): string {
  return attrKeys
    .map(k => [k, (values[k] ?? '').trim()] as const)
    .filter(([, v]) => v !== '')
    .map(([k, v]) => `${k}：${v}`)
    .join('，');
}

/* ------------------------------------------------------------------ */
/* 新增模版 Dialog                                                     */
/* 仅在 Inline 组件里点击「新增模版」时打开。                            */
/* ------------------------------------------------------------------ */

export interface ImageDescriptionTemplateCreateDialogProps {
  open: boolean;
  onClose: () => void;
  kbId: string;
  category: string;
  attrKeys: string[];
  /** 保存成功后回调，参数是渲染好的 K-V 描述文本和新建的模版对象 */
  onCreated: (description: string, item: ImageDescriptionTemplate) => void;
}

export const ImageDescriptionTemplateCreateDialog = ({
  open,
  onClose,
  kbId,
  category,
  attrKeys,
  onCreated,
}: ImageDescriptionTemplateCreateDialogProps) => {
  const [name, setName] = useState('');
  const [values, setValues] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      setName('');
      setValues({});
    }
  }, [open]);

  const handleSave = async () => {
    if (!kbId || !category) {
      message.error('请先选择品类');
      return;
    }
    const trimmed = name.trim();
    if (!trimmed) {
      message.error('请填写模版名称');
      return;
    }
    if (attrKeys.length === 0) {
      message.error('该品类未配置属性维护，无法创建结构化模版');
      return;
    }
    const cleanValues: Record<string, string> = {};
    attrKeys.forEach(k => {
      const v = (values[k] || '').trim();
      if (v) cleanValues[k] = v;
    });
    if (Object.keys(cleanValues).length === 0) {
      message.error('请至少填写一个属性值');
      return;
    }
    setSaving(true);
    try {
      const res = await postApiV1ImageDescriptionTemplates({
        kb_id: kbId,
        category,
        name: trimmed,
        attributes: cleanValues,
      });
      const item = res?.item;
      const desc = renderAsDescription(attrKeys, cleanValues);
      message.success('模版已保存');
      if (item) onCreated(desc, item);
      onClose();
    } catch {
      // httpClient 已经全局 message.error
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth='sm' fullWidth>
      <DialogTitle>新增图片描述模版 — {category}</DialogTitle>
      <DialogContent>
        <Stack gap={1.5} sx={{ pt: 0.5 }}>
          <Typography variant='body2' color='text.secondary'>
            按品类的「属性维护」逐项填值，保存后会自动应用到当前图片，并在以后选择时复用。
          </Typography>
          <TextField
            label='模版名称'
            size='small'
            fullWidth
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder='例如：A 系列默认描述'
          />
          {attrKeys.map(k => (
            <TextField
              key={k}
              label={k}
              size='small'
              fullWidth
              value={values[k] ?? ''}
              onChange={e =>
                setValues(prev => ({ ...prev, [k]: e.target.value }))
              }
            />
          ))}
        </Stack>
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2 }}>
        <Button onClick={onClose}>取消</Button>
        <Button
          variant='contained'
          onClick={handleSave}
          disabled={saving || attrKeys.length === 0}
        >
          {saving ? '保存中…' : '保存并应用'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

/* ------------------------------------------------------------------ */
/* 内联组件：品类下拉 + 该品类模版列表 + 新增模版按钮                    */
/* 直接渲染在 tiptap 图片浮窗内。                                       */
/* ------------------------------------------------------------------ */

export interface InlineImageDescriptionTemplateProps {
  kbId: string;
  /** 当前 attrs.title。用于在已经匹配某模版时把它高亮（按渲染文本对比）。 */
  value?: string;
  /** 用户在内联组件里选中/新增模版后，写回 popover 的 editTitle */
  onChange: (description: string) => void;
}

export const InlineImageDescriptionTemplate = ({
  kbId,
  value,
  onChange,
}: InlineImageDescriptionTemplateProps) => {
  const [categories, setCategories] = useState<CategoryPromptItem[]>([]);
  const [categoryLoading, setCategoryLoading] = useState(false);
  const [selectedCategory, setSelectedCategory] = useState('');

  const [templates, setTemplates] = useState<ImageDescriptionTemplate[]>([]);
  const [templatesLoading, setTemplatesLoading] = useState(false);

  const [createOpen, setCreateOpen] = useState(false);

  const currentCategory = useMemo(
    () => categories.find(c => c.name.trim() === selectedCategory),
    [categories, selectedCategory],
  );
  const attrKeys = useMemo(
    () => splitAttrKeys(currentCategory?.attributes),
    [currentCategory],
  );

  const loadCategories = useCallback(async () => {
    if (!kbId) return;
    setCategoryLoading(true);
    try {
      const res = await getApiV1CategoryPrompts({ id: kbId });
      const list = (res?.items || []).filter(it => (it.name || '').trim());
      setCategories(list);
    } catch {
      setCategories([]);
    } finally {
      setCategoryLoading(false);
    }
  }, [kbId]);

  const loadTemplates = useCallback(
    async (category: string) => {
      if (!kbId || !category) {
        setTemplates([]);
        return;
      }
      setTemplatesLoading(true);
      try {
        const res = await getApiV1ImageDescriptionTemplates({
          kb_id: kbId,
          category,
        });
        setTemplates(res?.items || []);
      } catch {
        setTemplates([]);
      } finally {
        setTemplatesLoading(false);
      }
    },
    [kbId],
  );

  useEffect(() => {
    void loadCategories();
  }, [loadCategories]);

  useEffect(() => {
    void loadTemplates(selectedCategory);
  }, [selectedCategory, loadTemplates]);

  const handlePickTemplate = (tmpl: ImageDescriptionTemplate) => {
    const desc = renderAsDescription(attrKeys, tmpl.attributes || {});
    if (!desc) {
      message.warning('该模版没有可用的属性值');
      return;
    }
    onChange(desc);
  };

  return (
    <Stack gap={1} sx={{ minHeight: 0 }}>
      {categoryLoading ? (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.5 }}>
          <CircularProgress size={14} />
          <Typography variant='caption' color='text.secondary'>
            加载品类…
          </Typography>
        </Box>
      ) : categories.length === 0 ? (
        <Alert severity='warning' sx={{ py: 0.25 }}>
          当前知识库还没有维护品类，请先到「提示词管理」配置。
        </Alert>
      ) : (
        <Select
          size='small'
          fullWidth
          displayEmpty
          value={selectedCategory}
          onChange={e => setSelectedCategory(e.target.value)}
        >
          <MenuItem value=''>
            <em>请选择品类</em>
          </MenuItem>
          {categories.map(c => (
            <MenuItem key={c.id || c.name} value={c.name}>
              {c.name}
            </MenuItem>
          ))}
        </Select>
      )}

      {selectedCategory && attrKeys.length === 0 && (
        <Alert severity='info' sx={{ py: 0.25, fontSize: '0.75rem' }}>
          品类「{selectedCategory}」未配置「属性维护」，无法新增结构化模版。
        </Alert>
      )}

      {selectedCategory && (
        <Box
          sx={{
            maxHeight: 180,
            overflow: 'auto',
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 1,
            bgcolor: 'background.paper',
          }}
        >
          {templatesLoading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
              <CircularProgress size={18} />
            </Box>
          ) : templates.length === 0 ? (
            <Box sx={{ p: 1.25 }}>
              <Typography variant='caption' color='text.secondary'>
                该品类下还没有模版，点击下方「+ 新增模版」创建。
              </Typography>
            </Box>
          ) : (
            <List dense disablePadding>
              {templates.map(t => {
                const preview = renderAsDescription(
                  attrKeys,
                  t.attributes || {},
                );
                const hit = !!preview && preview === (value || '').trim();
                return (
                  <ListItemButton
                    key={t.id}
                    selected={hit}
                    onClick={() => handlePickTemplate(t)}
                    sx={{ py: 0.5 }}
                  >
                    <ListItemText
                      primaryTypographyProps={{
                        fontSize: '0.8125rem',
                        fontWeight: 500,
                      }}
                      secondaryTypographyProps={{
                        fontSize: '0.7rem',
                        sx: {
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                        },
                      }}
                      primary={t.name}
                      secondary={preview || '（暂无属性值）'}
                    />
                  </ListItemButton>
                );
              })}
            </List>
          )}
        </Box>
      )}

      <Button
        size='small'
        variant='outlined'
        onClick={() => setCreateOpen(true)}
        disabled={!selectedCategory || attrKeys.length === 0}
        sx={{ alignSelf: 'flex-start' }}
      >
        + 新增模版
      </Button>

      <ImageDescriptionTemplateCreateDialog
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        kbId={kbId}
        category={selectedCategory}
        attrKeys={attrKeys}
        onCreated={desc => {
          onChange(desc);
          void loadTemplates(selectedCategory);
        }}
      />
    </Stack>
  );
};

export default InlineImageDescriptionTemplate;
