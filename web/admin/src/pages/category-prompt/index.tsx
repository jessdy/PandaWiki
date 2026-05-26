import Card from '@/components/Card';
import {
  CategoryAttributeSpec,
  CategoryPromptItem,
  getApiV1CategoryPrompts,
  putApiV1CategoryPrompts,
} from '@/request/CategoryPrompt';
import { useAppSelector } from '@/store';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import {
  Box,
  Button,
  IconButton,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import { useCallback, useEffect, useState } from 'react';
import { v4 as uuidv4 } from 'uuid';
import AttributeSpecsEditor from './AttributeSpecsEditor';
import MethodRulesSection from './MethodRulesSection';

interface EditableCategory extends CategoryPromptItem {
  // 编辑态用结构化属性；保存时由它派生 attributes 字符串供后端兼容
  attribute_specs: CategoryAttributeSpec[];
}

const emptyRow = (): EditableCategory => ({
  id: '',
  name: '',
  content: '',
  attributes: '',
  attribute_specs: [],
});

/** 兼容旧数据：specs 为空时从 attributes 字符串升级出无枚举的 specs。 */
function ensureSpecs(item: CategoryPromptItem): CategoryAttributeSpec[] {
  if (item.attribute_specs && item.attribute_specs.length > 0) {
    return item.attribute_specs.map(s => ({
      name: (s.name || '').trim(),
      values: (s.values || []).map(v => v.trim()).filter(Boolean),
    }));
  }
  const raw = (item.attributes || '').trim();
  if (!raw) return [];
  return raw
    .replace(/\uff0c/g, ',')
    .split(',')
    .map(s => s.trim())
    .filter(Boolean)
    .map(name => ({ name, values: [] }));
}

const CategoryPromptPage = () => {
  const { kb_id } = useAppSelector(s => s.config);
  const [items, setItems] = useState<EditableCategory[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    if (!kb_id) return;
    setLoading(true);
    try {
      const res = await getApiV1CategoryPrompts({ id: kb_id });
      const raw = res?.items?.length ? res.items : [];
      const list: EditableCategory[] = raw.map(r => ({
        ...r,
        attribute_specs: ensureSpecs(r),
      }));
      setItems(list.length > 0 ? list : [emptyRow()]);
    } catch {
      setItems([emptyRow()]);
    } finally {
      setLoading(false);
    }
  }, [kb_id]);

  useEffect(() => {
    load();
  }, [load]);

  const onSave = async () => {
    if (!kb_id) {
      message.error('请先选择知识库');
      return;
    }
    for (const it of items) {
      const n = it.name.trim();
      const c = it.content.trim();
      if (!n && c) {
        message.error('请为已填写提示词的条目填写品类名，或清空该条提示词');
        return;
      }
      if (n && !c) {
        message.error(`品类「${n}」的提示词不能为空`);
        return;
      }
    }
    setSaving(true);
    try {
      const payload: CategoryPromptItem[] = items
        .filter(it => it.name.trim() !== '')
        .map(it => ({
          id: it.id,
          name: it.name,
          content: it.content,
          attribute_specs: it.attribute_specs.filter(s => s.name.trim() !== ''),
          // attributes 字符串由后端从 specs 派生（双写），这里不提交避免拉扯
        }));
      await putApiV1CategoryPrompts({ kb_id, items: payload });
      message.success('保存成功');
      await load();
    } finally {
      setSaving(false);
    }
  };

  const addRow = () => {
    setItems(prev => [...prev, { ...emptyRow(), id: uuidv4() }]);
  };

  const removeRow = (index: number) => {
    setItems(prev => {
      const next = prev.filter((_, i) => i !== index);
      return next.length > 0 ? next : [emptyRow()];
    });
  };

  if (!kb_id) {
    return (
      <Card>
        <Typography sx={{ p: 2 }} color='text.secondary'>
          请先选择知识库
        </Typography>
      </Card>
    );
  }

  return (
    <Card>
      <Stack sx={{ p: 2 }} gap={2}>
        <Box>
          <Typography variant='h6' sx={{ fontWeight: 600 }}>
            提示词
          </Typography>
          <Typography variant='body2' color='text.secondary' sx={{ mt: 0.5 }}>
            按品类维护提示词与「属性维护」。属性现支持配置枚举值：前台工作模式识别属性时会约束在枚举内，用户调整属性也只能从枚举里选。
            保存时仅保留已填写品类名的条目。图片类文档生成摘要时，会先判断是否属于某一品类：命中则按对应提示词写摘要；未命中则对画面做细致客观描述。
          </Typography>
          <Typography variant='body2' color='text.secondary' sx={{ mt: 0.5 }}>
            前台「工作模式」按以下顺序识别物品：识别品类 → 按枚举抽取属性 →
            在「开封方法规则」中查表 → 命中即生成方法卡片并链接到对应文档；
            若未命中规则则回退到既有的「目录内向量检索 + 按文档侧
            meta.attributes 收敛」流程。
          </Typography>
        </Box>

        {loading ? (
          <Typography color='text.secondary'>加载中…</Typography>
        ) : (
          <Stack gap={2}>
            {items.map((row, index) => (
              <Stack
                key={row.id || `row-${index}`}
                gap={2}
                sx={{
                  p: 2,
                  borderRadius: 1,
                  border: '1px solid',
                  borderColor: 'divider',
                }}
              >
                <Stack
                  direction={{ xs: 'column', md: 'row' }}
                  gap={2}
                  alignItems={{ md: 'flex-start' }}
                >
                  <TextField
                    label='品类名'
                    size='small'
                    value={row.name}
                    onChange={e => {
                      const v = e.target.value;
                      setItems(prev =>
                        prev.map((it, i) =>
                          i === index ? { ...it, name: v } : it,
                        ),
                      );
                    }}
                    sx={{ minWidth: { md: 200 }, flexShrink: 0 }}
                  />
                  <Box sx={{ flex: 1, minWidth: 0 }}>
                    <TextField
                      label='提示词'
                      size='small'
                      fullWidth
                      multiline
                      minRows={4}
                      value={row.content}
                      onChange={e => {
                        const v = e.target.value;
                        setItems(prev =>
                          prev.map((it, i) =>
                            i === index ? { ...it, content: v } : it,
                          ),
                        );
                      }}
                    />
                  </Box>
                  <IconButton
                    aria-label='删除品类'
                    color='error'
                    onClick={() => removeRow(index)}
                    sx={{ alignSelf: { xs: 'flex-end', md: 'center' } }}
                  >
                    <DeleteOutlineIcon />
                  </IconButton>
                </Stack>

                <AttributeSpecsEditor
                  value={row.attribute_specs}
                  onChange={specs =>
                    setItems(prev =>
                      prev.map((it, i) =>
                        i === index ? { ...it, attribute_specs: specs } : it,
                      ),
                    )
                  }
                />

                {row.name.trim() && row.id && (
                  <MethodRulesSection
                    kbId={kb_id}
                    category={row.name.trim()}
                    attrSpecs={row.attribute_specs.filter(
                      s => s.name.trim() !== '',
                    )}
                  />
                )}
                {row.name.trim() && !row.id && (
                  <Typography variant='caption' color='text.tertiary'>
                    💡 新品类保存后才能开始添加「开封方法规则」。
                  </Typography>
                )}
              </Stack>
            ))}
          </Stack>
        )}

        <Stack direction='row' gap={2} flexWrap='wrap'>
          <Button variant='outlined' onClick={addRow} disabled={loading}>
            添加品类
          </Button>
          <Button
            variant='contained'
            onClick={onSave}
            disabled={loading || saving}
          >
            {saving ? '保存中…' : '保存'}
          </Button>
        </Stack>
      </Stack>
    </Card>
  );
};

export default CategoryPromptPage;
