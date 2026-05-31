'use client';

import { WelcomeFooter } from '@/components/footer';
import { WelcomeHeader } from '@/components/header';
import { useStore } from '@/provider';
import { createComponentStyleOverrides } from '@/theme';
import { THEME_TO_PALETTE } from '@panda-wiki/themes/constants';
import { ThemeProvider } from '@ctzhian/ui';
import { Stack, createTheme, Box } from '@mui/material';
import { useMemo } from 'react';

/**
 * 「疑难咨询」页的统一外壳：顶部 WelcomeHeader + 底部 WelcomeFooter + 居中内容区。
 * 主题色与首页保持一致，复用 web_app_landing_theme 配置。
 */
const ConsultChrome = ({ children }: { children: React.ReactNode }) => {
  const { kbDetail } = useStore();

  const theme = useMemo(() => {
    // @ts-ignore
    const themeMode = kbDetail?.settings?.web_app_landing_theme?.name || 'blue';
    return createTheme({
      cssVariables: { cssVarPrefix: 'welcome' },
      palette:
        THEME_TO_PALETTE[themeMode]?.palette ||
        THEME_TO_PALETTE['blue'].palette,
      typography: {
        fontFamily: 'var(--font-gilory), PingFang SC, sans-serif',
      },
      components: createComponentStyleOverrides(true),
    });
    // @ts-ignore
  }, [kbDetail?.settings?.web_app_landing_theme?.name]);

  return (
    <ThemeProvider theme={theme}>
      <Stack
        justifyContent='space-between'
        sx={{ minHeight: '100vh', bgcolor: 'background.default' }}
      >
        <WelcomeHeader />
        <Stack sx={{ flex: 1, alignItems: 'center' }}>
          <Box
            sx={{
              width: 800,
              maxWidth: '100%',
              flex: 1,
              px: { xs: 2, sm: 3 },
            }}
          >
            {children}
          </Box>
        </Stack>
        <WelcomeFooter />
      </Stack>
    </ThemeProvider>
  );
};

export default ConsultChrome;
