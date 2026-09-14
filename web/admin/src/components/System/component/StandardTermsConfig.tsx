import {
  getApiV1SystemStandardTerms,
  putApiV1SystemStandardTerms,
  StandardTermItem,
} from '@/request/StandardTerm';
import { message, Modal } from '@ctzhian/ui';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  CircularProgress,
  IconButton,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useMemo,
  useState,
} from 'react';
import { v4 as uuidv4 } from 'uuid';

export interface StandardTermsConfigRef {
  hasUnsavedChanges: () => boolean;
  handleClose: () => void;
}

interface StandardTermsConfigProps {
  onCloseModal: () => void;
}

const cleanAliases = (aliases: string[]) =>
  Array.from(
    new Map(
      aliases
        .map(alias => alias.trim())
        .filter(Boolean)
        .map(alias => [alias.toLocaleLowerCase(), alias]),
    ).values(),
  );

const validateItems = (items: StandardTermItem[]): string => {
  if (items.length > 200) return '标准术语最多可配置 200 个';
  const standards = new Map<string, string>();
  for (const item of items) {
    const term = item.term.trim();
    if (!term) return '标准术语不能为空';
    const key = term.toLocaleLowerCase();
    if (standards.has(key)) return `标准术语“${term}”重复`;
    standards.set(key, term);
  }
  const aliases = new Map<string, string>();
  for (const item of items) {
    const term = item.term.trim();
    const values = cleanAliases(item.aliases);
    if (values.length === 0) return `标准术语“${term}”至少需要一个别名`;
    if (values.length > 50) return `标准术语“${term}”最多可配置 50 个别名`;
    for (const alias of values) {
      const key = alias.toLocaleLowerCase();
      if (standards.has(key)) {
        return `别名“${alias}”与标准术语“${standards.get(key)}”冲突`;
      }
      if (aliases.has(key)) {
        return `别名“${alias}”已属于标准术语“${aliases.get(key)}”`;
      }
      aliases.set(key, term);
    }
  }
  return '';
};

interface AliasEditorProps {
  value: string[];
  inputValue: string;
  onChange: (value: string[]) => void;
  onInputChange: (value: string) => void;
}

const AliasEditor = ({
  value,
  inputValue,
  onChange,
  onInputChange,
}: AliasEditorProps) => {
  const commitInput = () => {
    const pending = inputValue.trim();
    if (!pending) return;
    onChange(cleanAliases([...value, pending]));
    onInputChange('');
  };

  return (
    <Autocomplete
      multiple
      freeSolo
      size='small'
      options={[]}
      value={value}
      inputValue={inputValue}
      onInputChange={(_, next) => onInputChange(next)}
      onChange={(_, next) => onChange(cleanAliases(next as string[]))}
      onBlur={commitInput}
      renderInput={params => (
        <TextField
          {...params}
          label='别名'
          placeholder={value.length ? '' : '输入别名后按回车'}
          helperText='AI 将在培训模式提问中识别这些说法并映射为标准术语'
        />
      )}
    />
  );
};

const StandardTermsConfig = forwardRef<
  StandardTermsConfigRef,
  StandardTermsConfigProps
>(({ onCloseModal }, ref) => {
  const [items, setItems] = useState<StandardTermItem[]>([]);
  const [savedItems, setSavedItems] = useState<StandardTermItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const [pendingAliases, setPendingAliases] = useState<Record<string, string>>(
    {},
  );

  const dirty = useMemo(
    () =>
      JSON.stringify(items) !== JSON.stringify(savedItems) ||
      Object.values(pendingAliases).some(value => value.trim() !== ''),
    [items, pendingAliases, savedItems],
  );

  const load = async () => {
    setLoading(true);
    setLoadError(false);
    try {
      const result = await getApiV1SystemStandardTerms();
      const next = result?.items ?? [];
      setItems(next);
      setSavedItems(next);
      setPendingAliases({});
    } catch {
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleClose = () => {
    if (!dirty) {
      onCloseModal();
      return;
    }
    Modal.confirm({
      title: '有未保存的标准术语',
      content: '关闭后本次修改将丢失，是否继续？',
      okText: '确认关闭',
      cancelText: '继续编辑',
      onOk: onCloseModal,
    });
  };

  useImperativeHandle(ref, () => ({
    hasUnsavedChanges: () => dirty,
    handleClose,
  }));

  const updateItem = (id: string, patch: Partial<StandardTermItem>) => {
    setItems(current =>
      current.map(item => (item.id === id ? { ...item, ...patch } : item)),
    );
  };

  const handleSave = async () => {
    const normalized = items.map(item => ({
      ...item,
      term: item.term.trim(),
      aliases: cleanAliases([...item.aliases, pendingAliases[item.id] ?? '']),
    }));
    const validationError = validateItems(normalized);
    if (validationError) {
      message.error(validationError);
      return;
    }
    setSaving(true);
    try {
      const result = await putApiV1SystemStandardTerms({ items: normalized });
      const saved = result?.items ?? normalized;
      setItems(saved);
      setSavedItems(saved);
      setPendingAliases({});
      message.success('标准术语已保存');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Stack
        alignItems='center'
        justifyContent='center'
        sx={{ minHeight: 360 }}
      >
        <CircularProgress size={28} />
      </Stack>
    );
  }

  if (loadError) {
    return (
      <Stack
        alignItems='center'
        justifyContent='center'
        gap={2}
        sx={{ minHeight: 360 }}
      >
        <Alert severity='error'>标准术语加载失败</Alert>
        <Button variant='outlined' onClick={load}>
          重新加载
        </Button>
      </Stack>
    );
  }

  return (
    <Stack gap={2.5}>
      <Box>
        <Typography variant='h6' sx={{ fontWeight: 600 }}>
          标准术语
        </Typography>
        <Typography variant='body2' color='text.secondary' sx={{ mt: 0.5 }}>
          该配置全局生效。培训模式会先用分析模型理解问题中的别名，再以标准术语进行知识库检索；用户原始问题不会被修改。
        </Typography>
      </Box>

      {items.length === 0 ? (
        <Box
          sx={{
            py: 6,
            textAlign: 'center',
            border: '1px dashed',
            borderColor: 'divider',
            borderRadius: 2,
          }}
        >
          <Typography color='text.secondary'>暂未配置标准术语</Typography>
          <Button
            startIcon={<AddIcon />}
            sx={{ mt: 1.5 }}
            onClick={() => setItems([{ id: uuidv4(), term: '', aliases: [] }])}
          >
            添加标准术语
          </Button>
        </Box>
      ) : (
        <Stack gap={1.5} sx={{ maxHeight: 520, overflowY: 'auto', pr: 0.5 }}>
          {items.map((item, index) => (
            <Stack
              key={item.id}
              direction={{ xs: 'column', md: 'row' }}
              gap={1.5}
              alignItems={{ md: 'flex-start' }}
              sx={{
                p: 2,
                border: '1px solid',
                borderColor: 'divider',
                borderRadius: 2,
                bgcolor: 'background.paper3',
              }}
            >
              <TextField
                size='small'
                label={`标准术语 ${index + 1}`}
                value={item.term}
                onChange={event =>
                  updateItem(item.id, { term: event.target.value })
                }
                placeholder='例如：聚对苯二甲酸乙二醇酯'
                inputProps={{ maxLength: 100 }}
                sx={{ width: { md: 260 }, flexShrink: 0 }}
              />
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <AliasEditor
                  value={item.aliases}
                  inputValue={pendingAliases[item.id] ?? ''}
                  onChange={aliases => updateItem(item.id, { aliases })}
                  onInputChange={value =>
                    setPendingAliases(current => ({
                      ...current,
                      [item.id]: value,
                    }))
                  }
                />
              </Box>
              <IconButton
                aria-label={`删除标准术语 ${item.term || index + 1}`}
                color='error'
                onClick={() => {
                  setItems(current =>
                    current.filter(currentItem => currentItem.id !== item.id),
                  );
                  setPendingAliases(current => {
                    const next = { ...current };
                    delete next[item.id];
                    return next;
                  });
                }}
              >
                <DeleteOutlineIcon />
              </IconButton>
            </Stack>
          ))}
        </Stack>
      )}

      <Stack direction='row' justifyContent='space-between'>
        <Button
          variant='outlined'
          startIcon={<AddIcon />}
          disabled={items.length >= 200}
          onClick={() =>
            setItems(current => [
              ...current,
              { id: uuidv4(), term: '', aliases: [] },
            ])
          }
        >
          添加标准术语
        </Button>
        <Button
          variant='contained'
          loading={saving}
          disabled={!dirty}
          onClick={handleSave}
        >
          保存设置
        </Button>
      </Stack>
    </Stack>
  );
});

StandardTermsConfig.displayName = 'StandardTermsConfig';

export default StandardTermsConfig;
