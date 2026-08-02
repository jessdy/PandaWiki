'use client';

import { ITreeItem, KBDetail, NodeListItem, WidgetInfo } from '@/assets/type';
import { useMediaQuery } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import {
  createContext,
  useContext,
  useEffect,
  useState,
  Dispatch,
  SetStateAction,
} from 'react';
import { GithubComChaitinPandaWikiProApiShareV1AuthInfoResp } from '@/request/pro/types';
import { DEMO_AUTH_INFO } from '@/utils/authInfo';

interface StoreContextType {
  authInfo?: GithubComChaitinPandaWikiProApiShareV1AuthInfoResp;
  widget?: WidgetInfo;
  kbDetail?: KBDetail;
  catalogShow?: boolean;
  tree?: ITreeItem[];
  themeMode?: 'light' | 'dark';
  mobile?: boolean;
  nodeList?: NodeListItem[];
  setNodeList?: (list: NodeListItem[]) => void;
  setTree?: Dispatch<SetStateAction<ITreeItem[] | undefined>>;
  setCatalogShow?: (value: boolean) => void;
  catalogWidth?: number;
  setCatalogWidth?: (value: number) => void;
  qaModalOpen?: boolean;
  setQaModalOpen?: (value: boolean) => void;
  chatSearchImages?: File[];
  setChatSearchImages?: Dispatch<SetStateAction<File[]>>;
  loginModalOpen?: boolean;
  setLoginModalOpen?: (value: boolean) => void;
  persistClientAuthInfo?: (
    info: GithubComChaitinPandaWikiProApiShareV1AuthInfoResp,
  ) => void;
  /** 退出或显式登出时清除本地 authInfo（含 localStorage） */
  clearClientAuthInfo?: () => void;
}

export const StoreContext = createContext<StoreContextType | undefined>(
  undefined,
);

export const useStore = () => {
  const context = useContext(StoreContext);
  if (!context) {
    throw new Error('useStore must be used within a StoreProvider');
  }
  return context;
};

export default function StoreProvider({
  children,
  ...props
}: StoreContextType & { children: React.ReactNode }) {
  const context = useContext(StoreContext) || {};
  const {
    widget = context.widget,
    kbDetail = context.kbDetail,
    themeMode = context.themeMode,
    nodeList: initialNodeList = context.nodeList || [],
    mobile = context.mobile,
    authInfo: authInfoProp = context.authInfo,
    tree: initialTree = context.tree || [],
  } = props;

  // 用 lazy initializer 在 client mount 的第一帧就把 localStorage 值拿到，
  // 避免「useEffect 读 localStorage」造成首帧 authInfo=undefined → 误判未登录的问题。
  // SSR 阶段 window 不存在，仍走 undefined；client 端 hydrate 时立即同步取值，无可见闪烁。
  const [localAuthInfo, setLocalAuthInfo] = useState<
    GithubComChaitinPandaWikiProApiShareV1AuthInfoResp | undefined
  >(() => {
    // demo 分支：无本地登录态时注入演示用户
    if (typeof window === 'undefined') return DEMO_AUTH_INFO;
    try {
      const raw = window.localStorage.getItem('authInfo');
      if (raw) {
        const parsed = JSON.parse(raw);
        if (
          parsed &&
          typeof parsed === 'object' &&
          Object.keys(parsed).length
        ) {
          return parsed;
        }
      }
    } catch {
      // ignore
    }
    try {
      window.localStorage.setItem('authInfo', JSON.stringify(DEMO_AUTH_INFO));
    } catch {
      // ignore
    }
    return DEMO_AUTH_INFO;
  });

  const authInfo = authInfoProp ?? localAuthInfo ?? DEMO_AUTH_INFO;

  const catalogSettings = kbDetail?.settings?.catalog_settings;

  const [catalogWidth, setCatalogWidth] = useState<number>(() => {
    return catalogSettings?.catalog_width || 260;
  });
  const [nodeList, setNodeList] = useState<NodeListItem[] | undefined>(
    initialNodeList,
  );
  const [tree, setTree] = useState<ITreeItem[] | undefined>(initialTree);
  const [qaModalOpen, setQaModalOpen] = useState(false);
  const [chatSearchImages, setChatSearchImages] = useState<File[]>([]);

  const persistClientAuthInfo = (
    info: GithubComChaitinPandaWikiProApiShareV1AuthInfoResp,
  ) => {
    try {
      window.localStorage.setItem('authInfo', JSON.stringify(info));
    } catch (_) {}
    setLocalAuthInfo(info);
  };

  const clearClientAuthInfo = () => {
    // demo 分支：退出后仍保留演示用户，避免被踢回登录
    try {
      window.localStorage.setItem('authInfo', JSON.stringify(DEMO_AUTH_INFO));
    } catch (_) {}
    setLocalAuthInfo(DEMO_AUTH_INFO);
  };

  const [catalogShow, setCatalogShow] = useState(
    catalogSettings?.catalog_visible !== 2,
  );
  const [isMobile, setIsMobile] = useState(mobile);
  const theme = useTheme();
  const mediaQueryResult = useMediaQuery(theme.breakpoints.down('lg'), {
    noSsr: true,
  });

  useEffect(() => {
    if (kbDetail) {
      setCatalogShow(catalogSettings?.catalog_visible !== 2);
    }
  }, [kbDetail]);

  useEffect(() => {
    const savedWidth = window.localStorage.getItem('CATALOG_WIDTH');
    if (Number(savedWidth) > 0) {
      setCatalogWidth(Number(savedWidth));
    }
  }, []);

  useEffect(() => {
    setIsMobile(mediaQueryResult);
  }, [mediaQueryResult]);

  return (
    <StoreContext.Provider
      value={{
        widget,
        kbDetail,
        themeMode,
        nodeList,
        catalogShow,
        setCatalogShow,
        mobile: isMobile,
        authInfo,
        setNodeList,
        catalogWidth,
        tree,
        setTree,
        setCatalogWidth: value => {
          setCatalogWidth(value);
          window.localStorage.setItem('CATALOG_WIDTH', value.toString());
        },
        qaModalOpen,
        setQaModalOpen,
        chatSearchImages,
        setChatSearchImages,
        loginModalOpen: false,
        setLoginModalOpen: () => {},
        persistClientAuthInfo,
        clearClientAuthInfo,
      }}
    >
      {children}
    </StoreContext.Provider>
  );
}
