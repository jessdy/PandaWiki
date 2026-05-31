import {
  CategoryAttributeSpec,
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

/** 解析品类的「属性维护」字符串（支持中英文逗号，兼容旧数据）。 */
function splitAttrKeys(raw?: string): string[] {
  if (!raw) return [];
  return raw
    .replace(/\uff0c/g, ',')
    .split(',')
    .map(s => s.trim())
    .filter(Boolean);
}

/** 结构化属性优先；无 specs 时从旧 attributes 字符串升级（无枚举）。 */
function resolveCategoryAttrSpecs(
  category?: CategoryPromptItem | null,
): CategoryAttributeSpec[] {
  if (!category) return [];
  const specs = category.attribute_specs;
  if (specs && specs.length > 0) {
    return specs
      .map(s => ({
        name: (s.name || '').trim(),
        values: (s.values || []).map(v => v.trim()).filter(Boolean),
      }))
      .filter(s => s.name !== '');
  }
  return splitAttrKeys(category.attributes).map(name => ({
    name,
    values: [] as string[],
  }));
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

/** 从已渲染的描述文本反解属性值（用于重新打开编辑时回填）。 */
function parseDescription(
  desc: string,
  attrKeys: string[],
): Record<string, string> {
  const result: Record<string, string> = {};
  const text = (desc || '').trim();
  if (!text || attrKeys.length === 0) return result;
  attrKeys.forEach(key => {
    const re = new RegExp(
      `${key.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}[：:]([^，,]+)`,
    );
    const m = text.match(re);
    if (m?.[1]) result[key] = m[1].trim();
  });
  return result;
}

/* ------------------------------------------------------------------ */
/* 属性值编辑区（新增模版 / 选中模版后共用）                              */
/* ------------------------------------------------------------------ */

function AttributeValueFields({
  attrSpecs,
  values,
  onChange,
}: {
  attrSpecs: CategoryAttributeSpec[];
  values: Record<string, string>;
  onChange: (values: Record<string, string>) => void;
}) {
  const setOne = (name: string, v: string) => {
    onChange({ ...values, [name]: v });
  };

  return (
    <>
      {attrSpecs.map(spec => {
        const cur = values[spec.name] ?? '';
        if (spec.values && spec.values.length > 0) {
          const options =
            cur && !spec.values.includes(cur)
              ? [cur, ...spec.values]
              : spec.values;
          return (
            <Select
              key={spec.name}
              size='small'
              fullWidth
              displayEmpty
              value={cur}
              onChange={e => setOne(spec.name, e.target.value)}
              renderValue={v =>
                v ? (
                  `${spec.name}：${v}`
                ) : (
                  <em style={{ opacity: 0.55 }}>{spec.name}（未选择）</em>
                )
              }
            >
              <MenuItem value=''>
                <em>清空</em>
              </MenuItem>
              {options.map(v => (
                <MenuItem key={v} value={v}>
                  {spec.name}：{v}
                </MenuItem>
              ))}
            </Select>
          );
        }
        return (
          <TextField
            key={spec.name}
            label={spec.name}
            size='small'
            fullWidth
            helperText='该属性未配置枚举值，可自由输入'
            value={cur}
            onChange={e => setOne(spec.name, e.target.value)}
            placeholder={`${spec.name}：`}
          />
        );
      })}
    </>
  );
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
  attrSpecs: CategoryAttributeSpec[];
  /** 保存成功后回调，参数是渲染好的 K-V 描述文本和新建的模版对象 */
  onCreated: (description: string, item: ImageDescriptionTemplate) => void;
}

export const ImageDescriptionTemplateCreateDialog = ({
  open,
  onClose,
  kbId,
  category,
  attrSpecs,
  onCreated,
}: ImageDescriptionTemplateCreateDialogProps) => {
  const attrKeys = useMemo(
    () => attrSpecs.map(s => s.name).filter(Boolean),
    [attrSpecs],
  );
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
            属性来自「提示词管理」的配置；已配置枚举值的属性请从下拉中选择，未配置枚举的可自由输入。
          </Typography>
          <TextField
            label='模版名称'
            size='small'
            fullWidth
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder='例如：A 系列默认描述'
          />
          <AttributeValueFields
            attrSpecs={attrSpecs}
            values={values}
            onChange={setValues}
          />
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
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>(
    null,
  );
  const [editValues, setEditValues] = useState<Record<string, string>>({});

  const currentCategory = useMemo(
    () => categories.find(c => c.name.trim() === selectedCategory),
    [categories, selectedCategory],
  );
  const attrSpecs = useMemo(
    () => resolveCategoryAttrSpecs(currentCategory),
    [currentCategory],
  );
  const attrKeys = useMemo(() => attrSpecs.map(s => s.name), [attrSpecs]);

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
    setSelectedTemplateId(null);
    setEditValues({});
  }, [selectedCategory, loadTemplates]);

  const applyEditValues = useCallback(
    (next: Record<string, string>) => {
      setEditValues(next);
      const desc = renderAsDescription(attrKeys, next);
      onChange(desc);
    },
    [attrKeys, onChange],
  );

  // 重新打开编辑时：若当前描述能解析出属性值，回填编辑区（不覆盖用户正在编辑的模版选中态）
  useEffect(() => {
    if (!selectedCategory || attrKeys.length === 0) return;
    const parsed = parseDescription(value || '', attrKeys);
    if (Object.keys(parsed).length === 0) return;

    setEditValues(prev => {
      const merged = attrKeys.reduce<Record<string, string>>((acc, k) => {
        acc[k] = parsed[k] ?? prev[k] ?? '';
        return acc;
      }, {});
      const same = attrKeys.every(k => (prev[k] ?? '') === merged[k]);
      return same ? prev : merged;
    });

    const matched = templates.find(t => {
      const preview = renderAsDescription(attrKeys, t.attributes || {});
      return preview && preview === (value || '').trim();
    });
    // 仅在有精确匹配时更新选中项；用户改属性后描述变化时保留原选中，避免编辑区消失
    if (matched?.id) {
      setSelectedTemplateId(matched.id);
    }
  }, [selectedCategory, attrKeys, value, templates]);

  const handlePickTemplate = (tmpl: ImageDescriptionTemplate) => {
    const attrs = tmpl.attributes || {};
    const hasValue = attrKeys.some(k => (attrs[k] ?? '').trim() !== '');
    if (!hasValue) {
      message.warning('该模版没有可用的属性值');
      return;
    }
    setSelectedTemplateId(tmpl.id ?? null);
    applyEditValues(
      attrKeys.reduce<Record<string, string>>((acc, k) => {
        acc[k] = (attrs[k] ?? '').trim();
        return acc;
      }, {}),
    );
  };

  const selectedTemplate = templates.find(t => t.id === selectedTemplateId);
  const showAttrEditor =
    selectedTemplateId !== null ||
    attrKeys.some(k => (editValues[k] ?? '').trim() !== '');

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
                const hit = t.id === selectedTemplateId;
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

      {selectedCategory && attrSpecs.length > 0 && showAttrEditor && (
        <Stack
          gap={1}
          sx={{
            p: 1.25,
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 1,
            bgcolor: 'background.paper3',
          }}
        >
          <Typography variant='caption' color='text.secondary'>
            {selectedTemplate
              ? `已选模版「${selectedTemplate.name}」，可按属性调整描述：`
              : '属性描述（可修改属性值）：'}
          </Typography>
          <AttributeValueFields
            attrSpecs={attrSpecs}
            values={editValues}
            onChange={applyEditValues}
          />
        </Stack>
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
        attrSpecs={attrSpecs}
        onCreated={(desc, item) => {
          onChange(desc);
          void loadTemplates(selectedCategory);
          if (item?.id) {
            setSelectedTemplateId(item.id);
            setEditValues(
              attrKeys.reduce<Record<string, string>>((acc, k) => {
                acc[k] = (item.attributes?.[k] ?? '').trim();
                return acc;
              }, {}),
            );
          }
        }}
      />
    </Stack>
  );
};

export default InlineImageDescriptionTemplate;
