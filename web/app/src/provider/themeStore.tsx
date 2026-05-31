'use client';
import { lightTheme } from '@/theme';
import { ThemeProvider } from '@ctzhian/ui';
import { createTheme } from '@mui/material';
import Cookies from 'js-cookie';
import { createContext, useContext, useEffect, useMemo } from 'react';

const ThemeContext = createContext<{
  themeMode: 'light' | 'dark';
  setThemeMode: (themeMode: 'light' | 'dark') => void;
}>({
  themeMode: 'light',
  setThemeMode: () => {},
});

export const useThemeStore = () => {
  return useContext(ThemeContext);
};

export const ThemeStoreProvider = ({
  children,
}: {
  themeMode?: 'light' | 'dark';
  children: React.ReactNode;
}) => {
  const themeMode = 'light' as const;
  const theme = useMemo(() => createTheme(lightTheme), []);

  useEffect(() => {
    Cookies.set('theme_mode', 'light', { expires: 365 * 10 });
  }, []);

  return (
    <ThemeContext.Provider value={{ themeMode, setThemeMode: () => {} }}>
      <ThemeProvider theme={theme}>{children}</ThemeProvider>
    </ThemeContext.Provider>
  );
};
