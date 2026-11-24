'use client';
import { useStore } from '@/provider';
import { useEffect, useMemo, useState } from 'react';
import { getImagePath } from '@/utils/getImagePath';
import { useBasePath } from '@/hooks';

import {
  Footer,
  WelcomeFooter as WelcomeFooterComponent,
} from '@panda-wiki/ui';

export const FooterProvider = ({
  showBrand = true,
  isDocPage = false,
  isWelcomePage = false,
}: {
  showBrand?: boolean;
  isDocPage?: boolean;
  isWelcomePage?: boolean;
}) => {
  const { mobile = false, catalogWidth, kbDetail } = useStore();
  const basePath = useBasePath();
  const docWidth = useMemo(() => {
    if (isWelcomePage) return 'full';
    return kbDetail?.settings?.theme_and_style?.doc_width || 'full';
  }, [kbDetail, isWelcomePage]);
  const footerSetting = kbDetail?.settings?.footer_settings;
  const customStyle = kbDetail?.settings?.web_app_custom_style;

  return (
    <>
      <Footer
        mobile={mobile}
        catalogWidth={catalogWidth}
        showBrand={showBrand}
        isDocPage={isDocPage}
        logo='https://release.baizhi.cloud/panda-wiki/icon.png'
        docWidth={docWidth}
        footerSetting={
          footerSetting
            ? {
                ...footerSetting,
                brand_logo: getImagePath(footerSetting?.brand_logo, basePath),
              }
            : undefined
        }
        customStyle={{
          ...customStyle,
          social_media_accounts: customStyle?.social_media_accounts?.map(
            (item: any) => ({
              ...item,
              icon: getImagePath(item.icon, basePath),
            }),
          ),
        }}
      />
      <AppVersion />
    </>
  );
};

function AppVersion() {
  const [version, setVersion] = useState('');
  useEffect(() => {
    fetch('/api/v1/system/info')
      .then(r => r.json())
      .then(body => {
        const data = body?.data ?? body;
        if (data?.version) setVersion(String(data.version));
      })
      .catch(() => undefined);
  }, []);
  if (!version) return null;
  return (
    <div
      style={{
        textAlign: 'center',
        fontSize: 12,
        opacity: 0.65,
        padding: '4px 0 8px',
      }}
    >
      版本 {version}
    </div>
  );
}

export const WelcomeFooter = ({
  showBrand = true,
}: {
  showBrand?: boolean;
}) => {
  const { mobile = false, catalogWidth, kbDetail } = useStore();
  const basePath = useBasePath();
  const footerSetting = kbDetail?.settings?.footer_settings;
  const customStyle = kbDetail?.settings?.web_app_custom_style;
  return (
    <>
      <WelcomeFooterComponent
        mobile={mobile}
        catalogWidth={catalogWidth}
        showBrand={showBrand}
        isDocPage={false}
        logo='https://release.baizhi.cloud/panda-wiki/icon.png'
        docWidth='full'
        footerSetting={
          footerSetting
            ? {
                ...footerSetting,
                brand_logo: getImagePath(footerSetting?.brand_logo, basePath),
              }
            : undefined
        }
        customStyle={{
          ...customStyle,
          social_media_accounts: customStyle?.social_media_accounts?.map(
            (item: any) => ({
              ...item,
              icon: getImagePath(item.icon, basePath),
            }),
          ),
        }}
      />
      <AppVersion />
    </>
  );
};

export default Footer;
