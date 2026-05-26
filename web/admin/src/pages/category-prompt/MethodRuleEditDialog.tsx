import { CategoryAttributeSpec } from '@/request/CategoryPrompt';
import { MethodRule } from '@/request/MethodRule';
import { getApiV1NodeList } from '@/request/Node';
import { DomainNodeListItemResp, DomainNodeType } from '@/request/types';
import {
  Autocomplete,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import { useCallback, useEffect, useMemo, useState } from 'react';

export interface MethodRuleEditDialogProps {
  open: boolean;
  onClose: () => void;
  kbId: string;
  category: string;
  attrSpecs: CategoryAttributeSpec[];
  /** 编辑场景传入 rule；新增传 null */
  rule: MethodRule | null;
  onSave: (rule: MethodRule) => void;
}

const blankConditions = (specs: CategoryAttributeSpec[]) => {
  const out: Record<string, string[]> = {};
  specs.forEach(s => {
    out[s.name] = [];
  });
  return out;
};

/**
 * 「开封方法规则」编辑器。每个属性按品类的枚举约束渲染：
 *   - values 非空 → Select 多选（严格只能选枚举里的值）
 *   - values 为空 → Autocomplete freeSolo（兼容老品类）
 *
 * 关联文档：Autocomplete 单选 + 远程搜索（debounce 300ms）。
 */
const MethodRuleEditDialog = ({
  open,
  onClose,
  kbId,
  category,
  attrSpecs,
  rule,
  onSave,
}: MethodRuleEditDialogProps) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [conditions, setConditions] = useState<Record<string, string[]>>({});
  const [nodeId, setNodeId] = useState('');
  const [nodeName, setNodeName] = useState('');
  const [nodeOptions, setNodeOptions] = useState<DomainNodeListItemResp[]>([]);
  const [nodeSearch, setNodeSearch] = useState('');
  const [nodeLoading, setNodeLoading] = useState(false);

  useEffect(() => {
    if (!open) return;
    if (rule) {
      setName(rule.name);
      setDescription(rule.description || '');
      // 用品类 specs 的字段顺序重排已有 conditions，避免属性顺序漂移
      const merged: Record<string, string[]> = blankConditions(attrSpecs);
      Object.entries(rule.conditions || {}).forEach(([k, v]) => {
        if (k in merged) merged[k] = Array.isArray(v) ? v : [];
      });
      setConditions(merged);
      setNodeId(rule.node_id || '');
    } else {
      setName('');
      setDescription('');
      setConditions(blankConditions(attrSpecs));
      setNodeId('');
      setNodeName('');
    }
  }, [open, rule, attrSpecs]);

  const loadNodes = useCallback(
    async (q: string) => {
      if (!kbId) return;
      setNodeLoading(true);
      try {
        const list = (await getApiV1NodeList({
          kb_id: kbId,
          ...(q.trim() ? { search: q.trim() } : {}),
        })) as DomainNodeListItemResp[] | undefined;
        const docs = (list || []).filter(
          n => n.type === DomainNodeType.NodeTypeDocument && n.id,
        );
        setNodeOptions(docs.slice(0, 120));
        // 编辑场景下，若当前 nodeId 在结果里，把 name 也补上展示
        if (nodeId) {
          const hit = docs.find(d => d.id === nodeId);
          if (hit?.name) setNodeName(hit.name);
        }
      } catch {
        setNodeOptions([]);
      } finally {
        setNodeLoading(false);
      }
    },
    [kbId, nodeId],
  );

  useEffect(() => {
    if (!open) return;
    void loadNodes('');
  }, [open, loadNodes]);

  const selectedNode = useMemo(() => {
    if (!nodeId) return null;
    const hit = nodeOptions.find(n => n.id === nodeId);
    if (hit) return hit;
    return {
      id: nodeId,
      name: nodeName || '(未在当前列表，但已记录 id)',
    } as DomainNodeListItemResp;
  }, [nodeId, nodeOptions, nodeName]);

  const handleSubmit = () => {
    const trimmedName = name.trim();
    if (!trimmedName) {
      message.error('请填写方法名');
      return;
    }
    if (!nodeId) {
      message.error('请选择关联文档');
      return;
    }
    // 过滤掉空条件项；至少需要一条有内容（避免规则永远命中）
    const cleanCond: Record<string, string[]> = {};
    let hasAny = false;
    Object.entries(conditions).forEach(([k, vs]) => {
      const arr = (vs || []).map(v => v.trim()).filter(Boolean);
      if (arr.length > 0) {
        cleanCond[k] = Array.from(new Set(arr));
        hasAny = true;
      }
    });
    if (!hasAny) {
      message.error('请至少为一个属性配置允许值（否则规则会无条件命中）');
      return;
    }
    onSave({
      id: rule?.id || '',
      category,
      name: trimmedName,
      description: description.trim(),
      conditions: cleanCond,
      node_id: nodeId,
      created_at: rule?.created_at,
      updated_at: rule?.updated_at,
    });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth='md' fullWidth>
      <DialogTitle>
        {rule ? '编辑开封方法规则' : '新增开封方法规则'} — {category}
      </DialogTitle>
      <DialogContent>
        <Stack gap={2} sx={{ pt: 0.5 }}>
          <TextField
            label='方法名'
            size='small'
            fullWidth
            required
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder='例如：易拉环开封'
          />
          <TextField
            label='方法说明（可选）'
            size='small'
            fullWidth
            multiline
            minRows={2}
            value={description}
            onChange={e => setDescription(e.target.value)}
            placeholder='前台卡片副标题展示'
          />

          <Box>
            <Typography
              variant='caption'
              color='text.secondary'
              sx={{ display: 'block', mb: 0.5 }}
            >
              属性条件：列出的属性必须全部命中（AND）；每个属性可选多个允许值（OR）。
              未列出的属性视为通配。
            </Typography>
            {attrSpecs.length === 0 ? (
              <Typography variant='body2' color='warning.main'>
                品类「{category}
                」未配置任何属性，请先到上方「属性维护」补充属性后再添加规则。
              </Typography>
            ) : (
              <Stack gap={1.5}>
                {attrSpecs.map(spec => {
                  const enumMode = (spec.values?.length ?? 0) > 0;
                  return (
                    <Stack
                      key={spec.name}
                      direction='row'
                      gap={1.5}
                      alignItems='flex-start'
                    >
                      <Typography
                        sx={{
                          minWidth: 96,
                          fontSize: '0.85rem',
                          pt: 1,
                          fontWeight: 500,
                          color: 'text.secondary',
                          textAlign: 'right',
                          flexShrink: 0,
                        }}
                      >
                        {spec.name}
                      </Typography>
                      <Box sx={{ flex: 1, minWidth: 0 }}>
                        {enumMode ? (
                          <Select
                            size='small'
                            multiple
                            fullWidth
                            displayEmpty
                            value={conditions[spec.name] || []}
                            onChange={e => {
                              const v = e.target.value as string[];
                              setConditions(prev => ({
                                ...prev,
                                [spec.name]: v,
                              }));
                            }}
                            renderValue={selected => {
                              const arr = selected as string[];
                              if (arr.length === 0) {
                                return (
                                  <Typography
                                    component='span'
                                    variant='body2'
                                    color='text.tertiary'
                                  >
                                    （留空 = 该属性不参与匹配）
                                  </Typography>
                                );
                              }
                              return (
                                <Stack
                                  direction='row'
                                  gap={0.5}
                                  flexWrap='wrap'
                                >
                                  {arr.map(v => (
                                    <Chip key={v} label={v} size='small' />
                                  ))}
                                </Stack>
                              );
                            }}
                          >
                            {spec.values!.map(v => (
                              <MenuItem key={v} value={v}>
                                {v}
                              </MenuItem>
                            ))}
                          </Select>
                        ) : (
                          <Autocomplete
                            size='small'
                            multiple
                            freeSolo
                            options={[]}
                            value={conditions[spec.name] || []}
                            onChange={(_, vs) =>
                              setConditions(prev => ({
                                ...prev,
                                [spec.name]: (vs as string[])
                                  .map(s => s.trim())
                                  .filter(Boolean),
                              }))
                            }
                            renderInput={params => (
                              <TextField
                                {...params}
                                placeholder='输入允许值后回车（该属性未配置枚举）'
                              />
                            )}
                          />
                        )}
                      </Box>
                    </Stack>
                  );
                })}
              </Stack>
            )}
          </Box>

          <Box>
            <Typography
              variant='caption'
              color='text.secondary'
              sx={{ display: 'block', mb: 0.5 }}
            >
              关联文档（命中后用户点卡片跳转此文档）
            </Typography>
            <Autocomplete
              size='small'
              fullWidth
              loading={nodeLoading}
              options={nodeOptions}
              value={selectedNode}
              getOptionLabel={n => n?.name || '(未命名)'}
              isOptionEqualToValue={(a, b) => a?.id === b?.id}
              onChange={(_, n) => {
                setNodeId(n?.id || '');
                setNodeName(n?.name || '');
              }}
              onInputChange={(_, v, reason) => {
                if (reason === 'input') {
                  setNodeSearch(v);
                  // 轻量节流：每次输入直接 load（节点数通常不大）
                  void loadNodes(v);
                }
              }}
              renderInput={params => (
                <TextField
                  {...params}
                  placeholder='搜索文档标题…'
                  helperText={
                    nodeId && !selectedNode?.name?.trim()
                      ? '当前关联的文档不在搜索结果里，请重新搜索或确认未被删除'
                      : undefined
                  }
                />
              )}
              filterOptions={x => x}
              noOptionsText={nodeSearch ? '未找到匹配文档' : '请输入关键词搜索'}
            />
          </Box>
        </Stack>
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2 }}>
        <Button onClick={onClose}>取消</Button>
        <Button variant='contained' onClick={handleSubmit}>
          {rule ? '保存' : '新增'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default MethodRuleEditDialog;
