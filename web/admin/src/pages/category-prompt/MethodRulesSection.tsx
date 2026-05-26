import { CategoryAttributeSpec } from '@/request/CategoryPrompt';
import {
  MethodRule,
  getApiV1MethodRules,
  putApiV1MethodRules,
} from '@/request/MethodRule';
import { getApiV1NodeList } from '@/request/Node';
import { DomainNodeListItemResp } from '@/request/types';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import EditIcon from '@mui/icons-material/Edit';
import {
  Box,
  Button,
  Chip,
  IconButton,
  Stack,
  Typography,
} from '@mui/material';
import { message } from '@ctzhian/ui';
import { useCallback, useEffect, useMemo, useState } from 'react';
import MethodRuleEditDialog from './MethodRuleEditDialog';

export interface MethodRulesSectionProps {
  kbId: string;
  category: string;
  attrSpecs: CategoryAttributeSpec[];
}

/**
 * 显示在「品类」卡片下方的「开封方法规则」区域。
 *
 * - 拉取当前 KB 全量规则，按 category 过滤展示
 * - 增/改/删都先在内存里改，然后整表 PUT（与现有 settings JSON 模型一致）
 * - 关联文档名通过一次性 GET /node/list 缓存查表
 */
const MethodRulesSection = ({
  kbId,
  category,
  attrSpecs,
}: MethodRulesSectionProps) => {
  const [allRules, setAllRules] = useState<MethodRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [nodeNameMap, setNodeNameMap] = useState<Map<string, string>>(
    new Map(),
  );

  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<MethodRule | null>(null);

  const rulesOfCategory = useMemo(
    () => allRules.filter(r => (r.category || '').trim() === category.trim()),
    [allRules, category],
  );

  const loadAll = useCallback(async () => {
    if (!kbId) return;
    setLoading(true);
    try {
      const res = await getApiV1MethodRules({ kb_id: kbId });
      setAllRules(res?.items || []);
    } catch {
      setAllRules([]);
    } finally {
      setLoading(false);
    }
  }, [kbId]);

  const loadNodeNames = useCallback(async () => {
    if (!kbId) return;
    try {
      const list = (await getApiV1NodeList({ kb_id: kbId })) as
        | DomainNodeListItemResp[]
        | undefined;
      const m = new Map<string, string>();
      (list || []).forEach(n => {
        if (n.id) m.set(n.id, n.name || '');
      });
      setNodeNameMap(m);
    } catch {
      // ignore
    }
  }, [kbId]);

  useEffect(() => {
    void loadAll();
    void loadNodeNames();
  }, [loadAll, loadNodeNames]);

  // 通用持久化方法：用 nextItems 替换当前 KB 的所有规则
  const persist = async (nextItems: MethodRule[]) => {
    try {
      await putApiV1MethodRules({ kb_id: kbId, items: nextItems });
      message.success('已保存规则');
      await loadAll();
    } catch {
      // httpClient 全局 message.error
    }
  };

  const handleSaveRule = (saved: MethodRule) => {
    const idx = allRules.findIndex(r => r.id && r.id === saved.id);
    const next =
      idx === -1
        ? [...allRules, saved]
        : allRules.map(r => (r.id === saved.id ? saved : r));
    void persist(next);
    setEditorOpen(false);
    setEditingRule(null);
  };

  const handleDelete = (rule: MethodRule) => {
    if (!window.confirm(`删除规则「${rule.name}」？`)) return;
    void persist(allRules.filter(r => r.id !== rule.id));
  };

  const renderConditionPreview = (rule: MethodRule) => {
    const entries = Object.entries(rule.conditions || {}).filter(
      ([, vs]) => Array.isArray(vs) && vs.length > 0,
    );
    if (entries.length === 0) {
      return (
        <Typography variant='caption' color='warning.main'>
          ⚠ 无任何属性条件（会无条件命中，建议补充）
        </Typography>
      );
    }
    return (
      <Stack direction='row' flexWrap='wrap' gap={0.5}>
        {entries.map(([k, vs]) => (
          <Chip
            key={k}
            size='small'
            label={
              vs.length === 1 ? `${k} = ${vs[0]}` : `${k} ∈ {${vs.join(', ')}}`
            }
            variant='outlined'
          />
        ))}
      </Stack>
    );
  };

  return (
    <Box
      sx={{
        mt: 1.5,
        p: 1.5,
        borderRadius: 1,
        border: '1px dashed',
        borderColor: 'divider',
      }}
    >
      <Stack
        direction='row'
        alignItems='center'
        justifyContent='space-between'
        sx={{ mb: 1 }}
      >
        <Typography variant='body2' sx={{ fontWeight: 600 }}>
          开封方法规则 ({rulesOfCategory.length})
        </Typography>
        <Button
          size='small'
          variant='outlined'
          onClick={() => {
            setEditingRule(null);
            setEditorOpen(true);
          }}
          disabled={attrSpecs.length === 0}
        >
          + 添加规则
        </Button>
      </Stack>

      {attrSpecs.length === 0 ? (
        <Typography variant='caption' color='warning.main'>
          请先在上方「属性维护」补充属性（建议带枚举值）后再添加规则。
        </Typography>
      ) : loading ? (
        <Typography variant='caption' color='text.secondary'>
          加载中…
        </Typography>
      ) : rulesOfCategory.length === 0 ? (
        <Typography variant='caption' color='text.secondary'>
          暂无规则。点击「+ 添加规则」配置「属性条件 → 关联文档」的映射。
        </Typography>
      ) : (
        <Stack gap={1}>
          {rulesOfCategory.map(rule => {
            const nodeName =
              nodeNameMap.get(rule.node_id) || rule.node_id || '(未关联文档)';
            return (
              <Stack
                key={rule.id}
                direction='row'
                alignItems='flex-start'
                gap={1}
                sx={{
                  p: 1.25,
                  borderRadius: 1,
                  bgcolor: 'background.paper',
                  border: '1px solid',
                  borderColor: 'divider',
                }}
              >
                <Stack sx={{ flex: 1, minWidth: 0 }} gap={0.5}>
                  <Typography variant='body2' sx={{ fontWeight: 600 }}>
                    {rule.name}
                  </Typography>
                  {rule.description && (
                    <Typography variant='caption' color='text.secondary'>
                      {rule.description}
                    </Typography>
                  )}
                  {renderConditionPreview(rule)}
                  <Typography variant='caption' color='text.tertiary'>
                    → 文档：
                    {nodeNameMap.has(rule.node_id) ? (
                      nodeName
                    ) : (
                      <Box component='span' sx={{ color: 'error.main' }}>
                        ⚠ 关联文档可能已被删除（{rule.node_id}）
                      </Box>
                    )}
                  </Typography>
                </Stack>
                <IconButton
                  size='small'
                  onClick={() => {
                    setEditingRule(rule);
                    setEditorOpen(true);
                  }}
                  aria-label='编辑'
                >
                  <EditIcon fontSize='small' />
                </IconButton>
                <IconButton
                  size='small'
                  color='error'
                  onClick={() => handleDelete(rule)}
                  aria-label='删除'
                >
                  <DeleteOutlineIcon fontSize='small' />
                </IconButton>
              </Stack>
            );
          })}
        </Stack>
      )}

      <MethodRuleEditDialog
        open={editorOpen}
        onClose={() => {
          setEditorOpen(false);
          setEditingRule(null);
        }}
        kbId={kbId}
        category={category}
        attrSpecs={attrSpecs}
        rule={editingRule}
        onSave={handleSaveRule}
      />
    </Box>
  );
};

export default MethodRulesSection;
