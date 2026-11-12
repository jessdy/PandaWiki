import { getApiV1SystemInfo } from '@/request/SystemInfo';
import { Stack, Typography } from '@mui/material';
import { useEffect, useState } from 'react';
import packageJson from '../../../package.json';

const Version = () => {
  const fallback = import.meta.env.VITE_APP_VERSION || packageJson.version;
  const [version, setVersion] = useState(fallback);

  useEffect(() => {
    getApiV1SystemInfo()
      .then(res => {
        if (res?.version) {
          setVersion(res.version);
        }
      })
      .catch(() => undefined);
  }, []);

  return (
    <Stack
      justifyContent={'center'}
      sx={{
        borderTop: '1px solid',
        borderColor: 'divider',
        pt: 1.5,
        mt: 1,
        color: 'text.secondary',
        fontSize: 12,
        textAlign: 'center',
      }}
    >
      <Typography variant='caption' color='text.secondary'>
        版本 {version}
      </Typography>
    </Stack>
  );
};

export default Version;
