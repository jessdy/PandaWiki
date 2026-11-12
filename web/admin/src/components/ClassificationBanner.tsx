import { getApiV1SystemInfo } from '@/request/SystemInfo';
import { Box } from '@mui/material';
import { useEffect, useState } from 'react';

const ClassificationBanner = () => {
  const [level, setLevel] = useState('');

  useEffect(() => {
    getApiV1SystemInfo()
      .then(res => {
        if (res?.classification?.enabled && res.classification.level) {
          setLevel(res.classification.level);
        }
      })
      .catch(() => undefined);
  }, []);

  if (!level) return null;

  return (
    <Box
      sx={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        zIndex: 2000,
        bgcolor: '#b71c1c',
        color: '#fff',
        textAlign: 'center',
        py: 0.5,
        fontSize: 13,
        fontWeight: 600,
        letterSpacing: 2,
      }}
    >
      {level}
    </Box>
  );
};

export default ClassificationBanner;
