import { Stack, Typography } from '@mui/material';
import packageJson from '../../../package.json';

const Version = () => {
  const curVersion = import.meta.env.VITE_APP_VERSION || packageJson.version;

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
        版本 {curVersion}
      </Typography>
    </Stack>
  );
};

export default Version;
