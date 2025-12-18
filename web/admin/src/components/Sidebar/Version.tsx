import HelpCenter from '@/assets/json/help-center.json';
import IconUpgrade from '@/assets/json/upgrade.json';
import LottieIcon from '@/components/LottieIcon';
import { Box, Stack, Tooltip } from '@mui/material';
import { useEffect, useState } from 'react';
import packageJson from '../../../package.json';
import AuthTypeModal from './AuthTypeModal';
import { useVersionInfo } from '@/hooks';

const Version = () => {
  const versionInfo = useVersionInfo();
  const curVersion = import.meta.env.VITE_APP_VERSION || packageJson.version;
  const [latestVersion, setLatestVersion] = useState<string | undefined>(
    undefined,
  );
  const [typeOpen, setTypeOpen] = useState(false);

  useEffect(() => {
    fetch('https://release.baizhi.cloud/panda-wiki/version.txt')
      .then(response => response.text())
      .then(data => {
        setLatestVersion(data);
      })
      .catch(error => {
        console.error(error);
        setLatestVersion('');
      });
  }, []);

  if (latestVersion === undefined) return null;

  return (
    <>
      <Stack
        justifyContent={'center'}
        gap={0.5}
        sx={{
          borderTop: '1px solid',
          borderColor: 'divider',
          pt: 2,
          mt: 1,
          cursor: 'pointer',
          color: 'text.primary',
          fontSize: 12,
        }}
        onClick={() => setTypeOpen(true)}
      ></Stack>
      <AuthTypeModal
        open={typeOpen}
        onClose={() => setTypeOpen(false)}
        latestVersion={latestVersion}
        curVersion={curVersion}
      />
    </>
  );
};

export default Version;
