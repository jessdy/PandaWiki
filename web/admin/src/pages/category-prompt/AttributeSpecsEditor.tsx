import { CategoryAttributeSpec } from '@/request/CategoryPrompt';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import {
  Autocomplete,
  Box,
  Button,
  IconButton,
  Stack,
  TextField,
  Typography,
} from '@mui/material';

export interface AttributeSpecsEditorProps {
  value: CategoryAttributeSpec[];
  onChange: (value: CategoryAttributeSpec[]) => void;
}

/**
 * 属性结构化编辑器：每行 = [属性名] + [枚举值多选 chip]，外加增删按钮。
 *
 * 关键约束：
 *   - 属性名 trim 后非空才有效；空行保存时会被后端剔除。
 *   - 枚举值用 Autocomplete + freeSolo + multiple，允许键入新值回车成 chip。
 *   - 枚举值列表为空表示「不限定枚举」（兼容老品类），下游 UI 仍会显示文本输入兜底。
 */
const AttributeSpecsEditor = ({
  value,
  onChange,
}: AttributeSpecsEditorProps) => {
  const update = (index: number, patch: Partial<CategoryAttributeSpec>) => {
    onChange(value.map((s, i) => (i === index ? { ...s, ...patch } : s)));
  };
  const remove = (index: number) => {
    onChange(value.filter((_, i) => i !== index));
  };
  const add = () => {
    onChange([...value, { name: '', values: [] }]);
  };

  return (
    <Stack gap={1}>
      <Typography variant='caption' color='text.secondary'>
        属性维护：每个属性可枚举若干允许值（用于前台实战模式的 Select
        约束、规则匹配、属性面板）。留空表示该属性不限定枚举（仅按属性名兜底）。
      </Typography>
      {value.length === 0 && (
        <Typography variant='caption' color='text.tertiary' sx={{ pl: 0.5 }}>
          暂无属性，点击下方「+ 添加属性」开始配置。
        </Typography>
      )}
      {value.map((spec, index) => (
        <Stack
          key={index}
          direction={{ xs: 'column', md: 'row' }}
          gap={1.5}
          alignItems={{ md: 'flex-start' }}
          sx={{
            p: 1.25,
            borderRadius: 1,
            border: '1px solid',
            borderColor: 'divider',
            bgcolor: 'background.paper3',
          }}
        >
          <TextField
            size='small'
            label='属性名'
            value={spec.name}
            onChange={e => update(index, { name: e.target.value })}
            sx={{ minWidth: { md: 160 }, flexShrink: 0 }}
            placeholder='例如：材质'
          />
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Autocomplete
              size='small'
              multiple
              freeSolo
              options={[]}
              value={spec.values ?? []}
              onChange={(_, newValues) => {
                const clean = (newValues as string[])
                  .map(v => v.trim())
                  .filter(Boolean);
                update(index, { values: Array.from(new Set(clean)) });
              }}
              renderInput={params => (
                <TextField
                  {...params}
                  label='允许的枚举值（输入后回车成 chip）'
                  placeholder={
                    (spec.values?.length ?? 0) > 0
                      ? ''
                      : '例如：马口铁、玻璃、塑料'
                  }
                />
              )}
            />
          </Box>
          <IconButton
            aria-label='删除属性'
            color='error'
            onClick={() => remove(index)}
            sx={{ alignSelf: { xs: 'flex-end', md: 'center' } }}
          >
            <DeleteOutlineIcon />
          </IconButton>
        </Stack>
      ))}
      <Box>
        <Button size='small' variant='outlined' onClick={add}>
          + 添加属性
        </Button>
      </Box>
    </Stack>
  );
};

export default AttributeSpecsEditor;
