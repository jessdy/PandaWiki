'use client';

import SearchIcon from '@mui/icons-material/Search';
import CloseIcon from '@mui/icons-material/Close';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import {
  Box,
  ClickAwayListener,
  Fab,
  IconButton,
  InputAdornment,
  Paper,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useParams } from 'next/navigation';
import {
  clearDocFindHighlights,
  DOC_FIND_ACTIVE_CLASS,
  DOC_FIND_MARK_CLASS,
  getDocFindRoot,
  highlightDocFindMatches,
  scrollToDocFindMatch,
} from './docPageFind';

interface DocPageFindProps {
  mobile?: boolean;
}

const DocPageFind = ({ mobile }: DocPageFindProps) => {
  const { id: docId } = useParams() || {};
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [matchIndex, setMatchIndex] = useState(0);
  const [matchCount, setMatchCount] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const matchesRef = useRef<HTMLElement[]>([]);

  const applyActiveMark = useCallback(
    (matches: HTMLElement[], index: number) => {
      matches.forEach((el, i) => {
        el.classList.toggle(DOC_FIND_ACTIVE_CLASS, i === index);
      });
      scrollToDocFindMatch(matches[index]);
    },
    [],
  );

  const runSearch = useCallback(
    (value: string) => {
      const matches = highlightDocFindMatches(value);
      matchesRef.current = matches;
      setMatchCount(matches.length);
      if (matches.length === 0) {
        setMatchIndex(0);
        return;
      }
      setMatchIndex(0);
      applyActiveMark(matches, 0);
    },
    [applyActiveMark],
  );

  const close = useCallback(() => {
    setOpen(false);
    setQuery('');
    setMatchIndex(0);
    setMatchCount(0);
    matchesRef.current = [];
    clearDocFindHighlights();
  }, []);

  const goTo = useCallback(
    (delta: number) => {
      const matches = matchesRef.current;
      if (matches.length === 0) return;
      const next = (matchIndex + delta + matches.length) % matches.length;
      setMatchIndex(next);
      applyActiveMark(matches, next);
    },
    [applyActiveMark, matchIndex],
  );

  useEffect(() => {
    close();
  }, [docId, close]);

  useEffect(() => {
    if (!open) return;
    const timer = window.setTimeout(() => inputRef.current?.focus(), 50);
    return () => window.clearTimeout(timer);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const timer = window.setTimeout(() => runSearch(query), 180);
    return () => window.clearTimeout(timer);
  }, [open, query, runSearch]);

  useEffect(() => {
    return () => clearDocFindHighlights();
  }, []);

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
        const root = getDocFindRoot();
        if (!root) return;
        e.preventDefault();
        setOpen(true);
      }
      if (!open) return;
      if (e.key === 'Escape') {
        e.preventDefault();
        close();
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        goTo(e.shiftKey ? -1 : 1);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [close, goTo, open]);

  return (
    <>
      <Box
        sx={{
          position: 'fixed',
          top: mobile ? 64 : 72,
          right: mobile ? 12 : 16,
          zIndex: 10001,
        }}
      >
        {open && (
          <ClickAwayListener onClickAway={close}>
            <Paper
              elevation={4}
              sx={{
                position: 'absolute',
                top: 0,
                right: mobile ? 0 : 52,
                width: mobile ? 'min(92vw, 320px)' : 320,
                p: 1.5,
                borderRadius: 2,
                border: '1px solid',
                borderColor: 'divider',
              }}
              onMouseDown={e => e.stopPropagation()}
            >
              <TextField
                inputRef={inputRef}
                size='small'
                fullWidth
                placeholder='在文档中查找…'
                value={query}
                onChange={e => setQuery(e.target.value)}
                InputProps={{
                  endAdornment: (
                    <InputAdornment position='end'>
                      <IconButton size='small' onClick={close} edge='end'>
                        <CloseIcon fontSize='small' />
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
                onKeyDown={e => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    goTo(e.shiftKey ? -1 : 1);
                  }
                }}
              />
              <Stack
                direction='row'
                alignItems='center'
                justifyContent='space-between'
                sx={{ mt: 1 }}
              >
                <Typography variant='caption' color='text.secondary'>
                  {query.trim()
                    ? matchCount > 0
                      ? `${matchIndex + 1} / ${matchCount}`
                      : '无匹配'
                    : '输入关键词'}
                </Typography>
                <Stack direction='row' spacing={0.5}>
                  <IconButton
                    size='small'
                    disabled={matchCount === 0}
                    onClick={() => goTo(-1)}
                    aria-label='上一处'
                  >
                    <KeyboardArrowUpIcon fontSize='small' />
                  </IconButton>
                  <IconButton
                    size='small'
                    disabled={matchCount === 0}
                    onClick={() => goTo(1)}
                    aria-label='下一处'
                  >
                    <KeyboardArrowDownIcon fontSize='small' />
                  </IconButton>
                </Stack>
              </Stack>
            </Paper>
          </ClickAwayListener>
        )}

        <Tooltip title='文档内查找 (Ctrl+F)' placement='left' arrow>
          <Fab
            size='small'
            color={open ? 'primary' : 'default'}
            onClick={() => (open ? close() : setOpen(true))}
            sx={{
              backgroundColor: open ? undefined : 'background.paper3',
              color: open ? undefined : 'text.primary',
              boxShadow: 2,
              '&:hover': {
                backgroundColor: open ? undefined : 'background.paper2',
              },
            }}
          >
            <SearchIcon />
          </Fab>
        </Tooltip>
      </Box>

      <Box
        component='style'
        dangerouslySetInnerHTML={{
          __html: `
            mark.${DOC_FIND_MARK_CLASS} {
              background: rgba(255, 213, 0, 0.45);
              color: inherit;
              padding: 0 1px;
              border-radius: 2px;
            }
            mark.${DOC_FIND_MARK_CLASS}.${DOC_FIND_ACTIVE_CLASS} {
              background: rgba(255, 152, 0, 0.55);
              outline: 2px solid rgba(255, 152, 0, 0.85);
            }
          `,
        }}
      />
    </>
  );
};

export default DocPageFind;
