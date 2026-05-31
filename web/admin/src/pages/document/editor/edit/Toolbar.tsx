import { EditorToolbar, UseTiptapReturn } from '@ctzhian/tiptap';
import GridViewOutlinedIcon from '@mui/icons-material/GridViewOutlined';
import { Box, CircularProgress } from '@mui/material';
import { IconTianjiawendang } from '@panda-wiki/icons';

interface ToolbarProps {
  editorRef: UseTiptapReturn;
  imageLayoutLoading?: boolean;
  /** 从知识库已有文档快速插入链接 */
  onInsertKbDocLink?: () => void;
  onImageLayout?: () => void;
}

const Toolbar = ({
  editorRef,
  imageLayoutLoading = false,
  onInsertKbDocLink,
  onImageLayout,
}: ToolbarProps) => {
  return (
    <Box
      sx={{
        width: 'auto',
        border: '1px solid',
        borderColor: 'divider',
        borderRadius: '10px',
        bgcolor: 'background.default',
        px: 0.5,
        mx: 1,
      }}
    >
      <EditorToolbar
        editor={editorRef.editor}
        menuInToolbarMore={[
          ...(onInsertKbDocLink
            ? [
                {
                  id: 'kb-doc-link',
                  label: '知识库文档链接',
                  icon: <IconTianjiawendang sx={{ fontSize: '1rem' }} />,
                  onClick: onInsertKbDocLink,
                },
              ]
            : []),
          {
            id: 'image-layout',
            label: imageLayoutLoading ? '排版中' : '图片一键排版',
            icon: imageLayoutLoading ? (
              <CircularProgress size={16} />
            ) : (
              <GridViewOutlinedIcon sx={{ fontSize: '1rem' }} />
            ),
            onClick: imageLayoutLoading ? undefined : onImageLayout,
          },
        ]}
      />
    </Box>
  );
};

export default Toolbar;
