'use client';
import { useBasePath } from '@/hooks';
import {
  MethodMatchView,
  postShareV1MethodRulesMatch,
} from '@/request/ShareMethodRule';
import {
  AttrPanelMethod,
  AttrPanelSpec,
  AttributePanelMeta,
} from '@/utils/attributePanelParse';
import {
  Box,
  Button,
  CircularProgress,
  FormControl,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
  alpha,
} from '@mui/material';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

export interface AttributePanelProps {
  /** assistant 消息里解析出的初始 meta（含品类、specs、AI 已识别、初始命中方法） */
  meta: AttributePanelMeta;
  /** 配色钩子；为 null 时使用默认 MUI 主题 */
  workChrome: {
    gold: string;
    goldBright: string;
    border: string;
    borderStrong: string;
    textPrimary: string;
    textSecondary: string;
    textMuted: string;
    bgRaised: string;
  } | null;
}

/** 把当前属性收集状态拼成形如 "材质=马口铁 · 尺寸=500ml" 的预览串。 */
function previewCollected(collected: Record<string, string>): string {
  const parts = Object.entries(collected)
    .filter(([, v]) => v && v.trim())
    .map(([k, v]) => `${k}=${v.trim()}`);
  return parts.join(' · ');
}

const AttributePanel: React.FC<AttributePanelProps> = ({
  meta,
  workChrome,
}) => {
  const basePath = useBasePath();

  // 当前 specs 优先用接口最新返回的（处理后端品类被改名等场景），fallback 用 meta 里的
  const [specs, setSpecs] = useState<AttrPanelSpec[]>(meta.specs);
  // 用户编辑后的属性值；初始值取 AI 识别结果
  const [collected, setCollected] = useState<Record<string, string>>(
    meta.collected || {},
  );
  // 当前命中方法列表
  const [methods, setMethods] = useState<MethodMatchView[]>(meta.methods);
  // 调用 /share/v1/method_rules/match 时的 loading 状态（用于 chip 旋转）
  const [matching, setMatching] = useState(false);

  // 用户手动调整过哪些 key，用于 UI 区分「AI 识别」vs「已修改」
  const userTouchedRef = useRef<Set<string>>(new Set());

  // 防抖联动：300ms 内多次 onChange 只触发一次后端查表
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const refreshMatch = useCallback(
    (next: Record<string, string>) => {
      if (debounceTimer.current) clearTimeout(debounceTimer.current);
      debounceTimer.current = setTimeout(() => {
        setMatching(true);
        postShareV1MethodRulesMatch({
          category: meta.category,
          collected: next,
        })
          .then(resp => {
            setMethods(resp?.methods || []);
            if (resp?.specs && resp.specs.length > 0) setSpecs(resp.specs);
          })
          .catch(() => {
            // 静默：保留旧 methods
          })
          .finally(() => setMatching(false));
      }, 300);
    },
    [meta.category],
  );

  useEffect(() => {
    return () => {
      if (debounceTimer.current) clearTimeout(debounceTimer.current);
    };
  }, []);

  const missing = useMemo(
    () => specs.filter(s => !(collected[s.name] || '').trim()),
    [specs, collected],
  );

  const handleChange = (key: string, value: string) => {
    userTouchedRef.current.add(key);
    const next = { ...collected, [key]: value };
    if (!value) delete next[key];
    setCollected(next);
    refreshMatch(next);
  };

  const handleClear = (key: string) => {
    userTouchedRef.current.add(key);
    const next = { ...collected };
    delete next[key];
    setCollected(next);
    refreshMatch(next);
  };

  // 配色兜底
  const accent = workChrome?.gold || '#1976d2';
  const accentBright = workChrome?.goldBright || '#1976d2';
  const border = workChrome?.border || 'rgba(0,0,0,0.12)';
  const borderStrong = workChrome?.borderStrong || 'rgba(0,0,0,0.3)';
  const textPrimary = workChrome?.textPrimary || 'rgba(0,0,0,0.87)';
  const textSecondary = workChrome?.textSecondary || 'rgba(0,0,0,0.6)';
  const textMuted = workChrome?.textMuted || 'rgba(0,0,0,0.45)';
  const bgRaised = workChrome?.bgRaised || '#fff';

  return (
    <Stack gap={1.5} sx={{ mb: 1.5 }}>
      {/* 标题 + 已识别品类 */}
      <Box
        sx={{
          p: 1.25,
          borderRadius: '10px',
          border: `1px solid ${borderStrong}`,
          backgroundColor: bgRaised,
        }}
      >
        <Stack
          direction='row'
          alignItems='center'
          justifyContent='space-between'
          sx={{ mb: 0.5 }}
        >
          <Typography
            variant='body2'
            sx={{
              fontSize: 12,
              fontWeight: 600,
              color: accentBright,
              letterSpacing: '0.04em',
            }}
          >
            工作模式 · 已识别品类「{meta.category}」
          </Typography>
          {matching && (
            <Stack direction='row' alignItems='center' gap={0.5}>
              <CircularProgress size={12} sx={{ color: accent }} />
              <Typography
                variant='caption'
                sx={{ fontSize: 11, color: textMuted }}
              >
                联动刷新中…
              </Typography>
            </Stack>
          )}
        </Stack>
        <Typography variant='body2' sx={{ fontSize: 12, color: textSecondary }}>
          {missing.length === 0
            ? '所有属性已采集。下方展示按规则匹配出的开封方法。'
            : `还需补全 ${missing.length} 个属性：${missing.map(s => s.name).join('、')}`}
        </Typography>
      </Box>

      {/* 属性面板：每个属性一行 Select / TextField */}
      <Box
        sx={{
          p: 1.25,
          borderRadius: '10px',
          border: `1px solid ${border}`,
          backgroundColor: bgRaised,
        }}
      >
        <Typography
          variant='caption'
          sx={{ fontSize: 11, color: textMuted, display: 'block', mb: 0.75 }}
        >
          属性补全（调整后下方方法卡片自动联动刷新）
        </Typography>
        <Stack gap={1}>
          {specs.length === 0 ? (
            <Typography variant='body2' sx={{ fontSize: 12, color: textMuted }}>
              该品类未配置属性。
            </Typography>
          ) : (
            specs.map(spec => {
              const value = collected[spec.name] || '';
              const touched = userTouchedRef.current.has(spec.name);
              const aiPicked =
                !touched &&
                value !== '' &&
                (meta.collected || {})[spec.name] === value;
              const enumMode = (spec.values?.length ?? 0) > 0;
              return (
                <Stack
                  key={spec.name}
                  direction='row'
                  alignItems='center'
                  gap={1}
                >
                  <Typography
                    sx={{
                      width: 90,
                      flexShrink: 0,
                      fontSize: 13,
                      fontWeight: 500,
                      color: textPrimary,
                      textAlign: 'right',
                    }}
                  >
                    {spec.name}
                  </Typography>
                  <Box sx={{ flex: 1, minWidth: 0 }}>
                    {enumMode ? (
                      <FormControl fullWidth size='small'>
                        <Select
                          value={value}
                          displayEmpty
                          onChange={e =>
                            handleChange(spec.name, e.target.value as string)
                          }
                          sx={{
                            fontSize: 13,
                            backgroundColor: bgRaised,
                            color: textPrimary,
                            '& .MuiOutlinedInput-notchedOutline': {
                              borderColor: value ? borderStrong : border,
                            },
                            '&:hover .MuiOutlinedInput-notchedOutline': {
                              borderColor: borderStrong,
                            },
                          }}
                        >
                          <MenuItem value=''>
                            <em style={{ color: textMuted }}>请选择</em>
                          </MenuItem>
                          {spec.values!.map(v => (
                            <MenuItem key={v} value={v} sx={{ fontSize: 13 }}>
                              {v}
                            </MenuItem>
                          ))}
                        </Select>
                      </FormControl>
                    ) : (
                      <TextField
                        size='small'
                        fullWidth
                        value={value}
                        onChange={e => handleChange(spec.name, e.target.value)}
                        placeholder='该属性未配置枚举'
                        sx={{
                          backgroundColor: bgRaised,
                          '& .MuiInputBase-input': { fontSize: 13 },
                        }}
                      />
                    )}
                  </Box>
                  <Box
                    sx={{
                      width: 80,
                      flexShrink: 0,
                      textAlign: 'left',
                    }}
                  >
                    {aiPicked ? (
                      <Typography
                        variant='caption'
                        sx={{
                          fontSize: 10,
                          px: 0.75,
                          py: 0.25,
                          borderRadius: '4px',
                          color: accentBright,
                          backgroundColor: alpha(accent, 0.1),
                          border: `1px solid ${borderStrong}`,
                        }}
                      >
                        AI 识别
                      </Typography>
                    ) : value ? (
                      <Stack direction='row' alignItems='center' gap={0.5}>
                        <Typography
                          variant='caption'
                          sx={{
                            fontSize: 10,
                            px: 0.75,
                            py: 0.25,
                            borderRadius: '4px',
                            color: '#2e7d32',
                            backgroundColor: 'rgba(46,125,50,0.08)',
                            border: '1px solid rgba(46,125,50,0.4)',
                          }}
                        >
                          已修改
                        </Typography>
                        <Typography
                          variant='caption'
                          sx={{
                            fontSize: 10,
                            cursor: 'pointer',
                            color: textMuted,
                            '&:hover': { color: accent },
                          }}
                          onClick={() => handleClear(spec.name)}
                        >
                          清空
                        </Typography>
                      </Stack>
                    ) : (
                      <Typography
                        variant='caption'
                        sx={{
                          fontSize: 10,
                          px: 0.75,
                          py: 0.25,
                          borderRadius: '4px',
                          color: '#d32f2f',
                          backgroundColor: 'rgba(211,47,47,0.08)',
                          border: '1px solid rgba(211,47,47,0.4)',
                        }}
                      >
                        待补全
                      </Typography>
                    )}
                  </Box>
                </Stack>
              );
            })
          )}
        </Stack>

        {meta.unrecognized && Object.keys(meta.unrecognized).length > 0 && (
          <Typography
            variant='caption'
            sx={{
              fontSize: 11,
              color: '#ed6c02',
              mt: 1,
              display: 'block',
            }}
          >
            ⚠ AI 在描述里看到了「
            {Object.entries(meta.unrecognized)
              .map(([k, v]) => `${k}=${v}`)
              .join('、')}
            」，但与配置的枚举不符；请手动选择。
          </Typography>
        )}
      </Box>

      {/* 方法卡片区 */}
      <MethodCards
        methods={methods}
        collected={collected}
        missing={missing}
        basePath={basePath}
        workChrome={workChrome}
        textPrimary={textPrimary}
        textSecondary={textSecondary}
        textMuted={textMuted}
        accent={accent}
        accentBright={accentBright}
        bgRaised={bgRaised}
        border={border}
        borderStrong={borderStrong}
      />
    </Stack>
  );
};

interface MethodCardsProps {
  methods: AttrPanelMethod[];
  collected: Record<string, string>;
  missing: AttrPanelSpec[];
  basePath: string;
  workChrome: AttributePanelProps['workChrome'];
  textPrimary: string;
  textSecondary: string;
  textMuted: string;
  accent: string;
  accentBright: string;
  bgRaised: string;
  border: string;
  borderStrong: string;
}

const MethodCards: React.FC<MethodCardsProps> = ({
  methods,
  collected,
  missing,
  basePath,
  textPrimary,
  textSecondary,
  textMuted,
  accent,
  accentBright,
  bgRaised,
  border,
  borderStrong,
}) => {
  if (methods.length === 0) {
    return (
      <Box
        sx={{
          p: 1.5,
          borderRadius: '10px',
          border: `1px dashed ${border}`,
          backgroundColor: bgRaised,
        }}
      >
        <Typography variant='body2' sx={{ fontSize: 12, color: textMuted }}>
          {missing.length > 0
            ? `补全属性后将自动匹配开封方法。当前已采集：${previewCollected(collected) || '（无）'}`
            : '当前属性组合下未配置匹配的开封方法，请联系管理员补充规则。'}
        </Typography>
      </Box>
    );
  }
  return (
    <Box>
      <Typography
        variant='caption'
        sx={{ fontSize: 11, color: textMuted, mb: 0.75, display: 'block' }}
      >
        可能的开封方法（共 {methods.length} 个）
      </Typography>
      <Stack
        direction='row'
        flexWrap='wrap'
        gap={1}
        sx={{ alignItems: 'stretch' }}
      >
        {methods.map(m => (
          <Box
            key={m.id}
            sx={{
              flex: '1 1 240px',
              minWidth: 200,
              maxWidth: 360,
              p: 1.25,
              borderRadius: '10px',
              border: `1px solid ${borderStrong}`,
              backgroundColor: bgRaised,
              display: 'flex',
              flexDirection: 'column',
              gap: 0.75,
            }}
          >
            <Typography
              variant='body2'
              sx={{ fontSize: 14, fontWeight: 600, color: textPrimary }}
            >
              🔓 {m.name}
            </Typography>
            {m.description && (
              <Typography
                variant='caption'
                sx={{
                  fontSize: 12,
                  color: textSecondary,
                  whiteSpace: 'pre-wrap',
                }}
              >
                {m.description}
              </Typography>
            )}
            <Typography
              variant='caption'
              sx={{ fontSize: 11, color: textMuted, mt: 'auto' }}
            >
              {m.node_name
                ? `参考文档：${m.node_name}`
                : `⚠ 关联文档可能已被删除（${m.node_id}）`}
            </Typography>
            <Button
              variant='outlined'
              size='small'
              disabled={!m.node_name}
              onClick={() => {
                if (!m.node_id) return;
                window.open(`${basePath}/node/${m.node_id}`, '_blank');
              }}
              sx={{
                fontSize: 12,
                color: accentBright,
                borderColor: borderStrong,
                '&:hover': {
                  borderColor: accent,
                  backgroundColor: alpha(accent, 0.06),
                },
              }}
            >
              查看完整文档 →
            </Button>
          </Box>
        ))}
      </Stack>
    </Box>
  );
};

export default AttributePanel;
