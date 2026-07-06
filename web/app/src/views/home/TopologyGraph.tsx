'use client';

import { NodeListItem } from '@/assets/type';
import { IconWenjian, IconWenjianjia } from '@panda-wiki/icons';
import AddIcon from '@mui/icons-material/Add';
import RemoveIcon from '@mui/icons-material/Remove';
import CenterFocusStrongIcon from '@mui/icons-material/CenterFocusStrong';
import KeyboardArrowRightIcon from '@mui/icons-material/KeyboardArrowRight';
import {
  Box,
  IconButton,
  Stack,
  Tooltip,
  Typography,
  alpha,
} from '@mui/material';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

const ROOT_ID = '__topology_root__';

const NODE_W = 208;
const NODE_H = 46;
const LEVEL_GAP = 72;
const ROW_GAP = 18;
const LEVEL_WIDTH = NODE_W + LEVEL_GAP;
const ROW_HEIGHT = NODE_H + ROW_GAP;

interface TopoNode {
  id: string;
  name: string;
  /** 0=知识库根 1=目录 2=文档 */
  type: 0 | 1 | 2;
  emoji?: string;
  position: number;
  children: TopoNode[];
}

interface PlacedNode {
  node: TopoNode;
  x: number;
  y: number;
  depth: number;
  hasChildren: boolean;
  expanded: boolean;
}

interface TopologyGraphProps {
  nodeList: NodeListItem[];
  rootName: string;
  onOpenNode: (id: string) => void;
  mobile?: boolean;
}

/** 从扁平节点列表构建"仅含被勾选节点"的拓扑树，未勾选的父级会被跳过、子节点上挂到最近的已勾选祖先。 */
const buildTopoTree = (
  nodeList: NodeListItem[],
  rootName: string,
): TopoNode => {
  const byId = new Map<string, NodeListItem>();
  nodeList.forEach(n => {
    if (n.id) byId.set(n.id, n);
  });

  const isMarked = (n?: NodeListItem) => !!n?.meta?.show_in_topology;

  const nearestIncludedParentId = (n: NodeListItem): string | null => {
    let pid = n.parent_id;
    const seen = new Set<string>();
    while (pid && !seen.has(pid)) {
      seen.add(pid);
      const p = byId.get(pid);
      if (!p) return null;
      if (isMarked(p)) return p.id;
      pid = p.parent_id;
    }
    return null;
  };

  const root: TopoNode = {
    id: ROOT_ID,
    name: rootName || '知识库',
    type: 0,
    position: 0,
    children: [],
  };

  const topoMap = new Map<string, TopoNode>();
  const marked = nodeList.filter(isMarked);
  marked.forEach(n => {
    topoMap.set(n.id, {
      id: n.id,
      name: n.name,
      type: (n.type === 1 ? 1 : 2) as 1 | 2,
      emoji: n.meta?.emoji || n.emoji || undefined,
      position: n.position ?? 0,
      children: [],
    });
  });

  marked.forEach(n => {
    const topo = topoMap.get(n.id)!;
    const parentId = nearestIncludedParentId(n);
    const parent = parentId ? topoMap.get(parentId) : root;
    (parent ?? root).children.push(topo);
  });

  const sortRec = (node: TopoNode) => {
    node.children.sort((a, b) => a.position - b.position);
    node.children.forEach(sortRec);
  };
  sortRec(root);

  return root;
};

const TopologyGraph = ({
  nodeList,
  rootName,
  onOpenNode,
  mobile = false,
}: TopologyGraphProps) => {
  const root = useMemo(
    () => buildTopoTree(nodeList, rootName),
    [nodeList, rootName],
  );

  const [expanded, setExpanded] = useState<Set<string>>(
    () => new Set([ROOT_ID]),
  );
  const [scale, setScale] = useState(1);
  const [translate, setTranslate] = useState({ x: 40, y: 40 });
  const didInitRef = useRef(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<{
    dragging: boolean;
    startX: number;
    startY: number;
    baseX: number;
    baseY: number;
    moved: boolean;
  }>({
    dragging: false,
    startX: 0,
    startY: 0,
    baseX: 0,
    baseY: 0,
    moved: false,
  });

  const containerHeight = mobile ? 420 : 560;

  // 计算横向层级树布局（根在左，逐层向右）
  const { placed, links, width, height } = useMemo(() => {
    const placedList: PlacedNode[] = [];
    const placedMap = new Map<string, PlacedNode>();
    let row = 0;

    const walk = (node: TopoNode, depth: number): number => {
      const isExpanded = expanded.has(node.id);
      const kids = node.children;
      let y: number;
      if (kids.length > 0 && isExpanded) {
        const childYs = kids.map(k => walk(k, depth + 1));
        y = (childYs[0] + childYs[childYs.length - 1]) / 2;
      } else {
        y = row * ROW_HEIGHT;
        row += 1;
      }
      const entry: PlacedNode = {
        node,
        x: depth * LEVEL_WIDTH,
        y,
        depth,
        hasChildren: kids.length > 0,
        expanded: isExpanded,
      };
      placedList.push(entry);
      placedMap.set(node.id, entry);
      return y;
    };

    walk(root, 0);

    const linkList: { id: string; d: string }[] = [];
    placedList.forEach(entry => {
      if (!entry.expanded) return;
      entry.node.children.forEach(child => {
        const c = placedMap.get(child.id);
        if (!c) return;
        const x1 = entry.x + NODE_W;
        const y1 = entry.y + NODE_H / 2;
        const x2 = c.x;
        const y2 = c.y + NODE_H / 2;
        const midX = (x1 + x2) / 2;
        linkList.push({
          id: `${entry.node.id}->${child.id}`,
          d: `M${x1},${y1} C${midX},${y1} ${midX},${y2} ${x2},${y2}`,
        });
      });
    });

    const maxX = placedList.reduce((m, e) => Math.max(m, e.x + NODE_W), 0);
    const maxY = placedList.reduce((m, e) => Math.max(m, e.y + NODE_H), 0);

    return {
      placed: placedList,
      links: linkList,
      width: maxX,
      height: Math.max(maxY, NODE_H),
    };
  }, [root, expanded]);

  // 初次布局后把根节点垂直居中
  useEffect(() => {
    if (didInitRef.current) return;
    const rootEntry = placed.find(p => p.node.id === ROOT_ID);
    if (!rootEntry) return;
    didInitRef.current = true;
    setTranslate({
      x: 40,
      y: Math.max(24, containerHeight / 2 - (rootEntry.y + NODE_H / 2)),
    });
  }, [placed, containerHeight]);

  const toggle = useCallback((id: string) => {
    setExpanded(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const handleNodeClick = useCallback(
    (entry: PlacedNode) => {
      if (dragRef.current.moved) return;
      if (entry.hasChildren) {
        toggle(entry.node.id);
        return;
      }
      if (entry.node.id !== ROOT_ID) {
        onOpenNode(entry.node.id);
      }
    },
    [toggle, onOpenNode],
  );

  const clampScale = (s: number) => Math.min(2, Math.max(0.4, s));

  const zoomBy = (factor: number) => {
    setScale(prev => clampScale(prev * factor));
  };

  const resetView = () => {
    const rootEntry = placed.find(p => p.node.id === ROOT_ID);
    setScale(1);
    setTranslate({
      x: 40,
      y: rootEntry
        ? Math.max(24, containerHeight / 2 - (rootEntry.y + NODE_H / 2))
        : 40,
    });
  };

  const onWheel = (e: React.WheelEvent) => {
    if (!e.ctrlKey && !e.metaKey) return; // 仅按住 Ctrl/⌘ 时缩放，避免影响页面滚动
    e.preventDefault();
    zoomBy(e.deltaY < 0 ? 1.1 : 0.9);
  };

  const onMouseDown = (e: React.MouseEvent) => {
    dragRef.current = {
      dragging: true,
      startX: e.clientX,
      startY: e.clientY,
      baseX: translate.x,
      baseY: translate.y,
      moved: false,
    };
  };

  useEffect(() => {
    const onMove = (e: MouseEvent) => {
      const d = dragRef.current;
      if (!d.dragging) return;
      const dx = e.clientX - d.startX;
      const dy = e.clientY - d.startY;
      if (Math.abs(dx) > 3 || Math.abs(dy) > 3) d.moved = true;
      setTranslate({ x: d.baseX + dx, y: d.baseY + dy });
    };
    const onUp = () => {
      const d = dragRef.current;
      d.dragging = false;
      // 让点击判定读到 moved 后再复位
      setTimeout(() => {
        dragRef.current.moved = false;
      }, 0);
    };
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
    return () => {
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
    };
  }, []);

  return (
    <Box
      ref={containerRef}
      onMouseDown={onMouseDown}
      onWheel={onWheel}
      sx={theme => ({
        position: 'relative',
        height: containerHeight,
        width: '100%',
        overflow: 'hidden',
        borderRadius: '16px',
        border: '1px solid',
        borderColor: alpha(theme.palette.primary.main, 0.12),
        bgcolor: alpha(theme.palette.primary.main, 0.02),
        cursor: dragRef.current.dragging ? 'grabbing' : 'grab',
        backgroundImage: `radial-gradient(${alpha(
          theme.palette.text.primary,
          0.06,
        )} 1px, transparent 1px)`,
        backgroundSize: '22px 22px',
        userSelect: 'none',
      })}
    >
      <Box
        sx={{
          position: 'absolute',
          top: 0,
          left: 0,
          transform: `translate(${translate.x}px, ${translate.y}px) scale(${scale})`,
          transformOrigin: '0 0',
          width,
          height,
        }}
      >
        <svg
          width={Math.max(width, 1)}
          height={Math.max(height, 1)}
          style={{ position: 'absolute', top: 0, left: 0, overflow: 'visible' }}
        >
          {links.map(link => (
            <path
              key={link.id}
              d={link.d}
              fill='none'
              stroke='var(--welcome-palette-primary-main, #1976d2)'
              strokeOpacity={0.35}
              strokeWidth={2}
            />
          ))}
        </svg>

        {placed.map(entry => (
          <TopologyNodeCard
            key={entry.node.id}
            entry={entry}
            onClick={() => handleNodeClick(entry)}
          />
        ))}
      </Box>

      <Stack
        direction='column'
        spacing={0.5}
        sx={{
          position: 'absolute',
          right: 12,
          bottom: 12,
          bgcolor: 'background.paper',
          borderRadius: '10px',
          boxShadow: '0 2px 8px rgba(0,0,0,0.12)',
          p: 0.5,
        }}
      >
        <Tooltip title='放大' placement='left' arrow>
          <IconButton size='small' onClick={() => zoomBy(1.2)}>
            <AddIcon fontSize='small' />
          </IconButton>
        </Tooltip>
        <Tooltip title='缩小' placement='left' arrow>
          <IconButton size='small' onClick={() => zoomBy(0.8)}>
            <RemoveIcon fontSize='small' />
          </IconButton>
        </Tooltip>
        <Tooltip title='复位' placement='left' arrow>
          <IconButton size='small' onClick={resetView}>
            <CenterFocusStrongIcon fontSize='small' />
          </IconButton>
        </Tooltip>
      </Stack>

      <Typography
        variant='caption'
        sx={{
          position: 'absolute',
          left: 12,
          bottom: 12,
          color: 'text.tertiary',
          bgcolor: alpha('#ffffff', 0.6),
          px: 1,
          borderRadius: '6px',
          pointerEvents: 'none',
        }}
      >
        点击目录逐层展开 · 拖拽平移 · Ctrl/⌘ + 滚轮缩放
      </Typography>
    </Box>
  );
};

const TopologyNodeCard = ({
  entry,
  onClick,
}: {
  entry: PlacedNode;
  onClick: () => void;
}) => {
  const { node, x, y, hasChildren, expanded } = entry;
  const isRoot = node.type === 0;
  const isFolder = node.type === 1;

  return (
    <Box
      onClick={onClick}
      sx={theme => ({
        position: 'absolute',
        left: x,
        top: y,
        width: NODE_W,
        height: NODE_H,
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 1.5,
        boxSizing: 'border-box',
        borderRadius: '10px',
        cursor: 'pointer',
        transition: 'box-shadow .15s, transform .15s, border-color .15s',
        border: '1px solid',
        borderColor: isRoot
          ? 'transparent'
          : alpha(theme.palette.primary.main, 0.25),
        bgcolor: isRoot
          ? theme.palette.primary.main
          : theme.palette.background.paper,
        color: isRoot ? theme.palette.primary.contrastText : 'text.primary',
        boxShadow: isRoot
          ? `0 4px 14px ${alpha(theme.palette.primary.main, 0.35)}`
          : '0 1px 4px rgba(0,0,0,0.08)',
        '&:hover': {
          transform: 'translateY(-1px)',
          boxShadow: `0 4px 14px ${alpha(theme.palette.primary.main, 0.28)}`,
          borderColor: alpha(theme.palette.primary.main, 0.5),
        },
      })}
    >
      <Box
        sx={{
          flexShrink: 0,
          width: 22,
          height: 22,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 16,
          lineHeight: '22px',
        }}
      >
        {node.emoji ? (
          <span>{node.emoji}</span>
        ) : isFolder || isRoot ? (
          <IconWenjianjia sx={{ fontSize: 18 }} />
        ) : (
          <IconWenjian sx={{ fontSize: 18 }} />
        )}
      </Box>
      <Typography
        sx={{
          flex: 1,
          minWidth: 0,
          fontSize: 14,
          fontWeight: isRoot ? 700 : 500,
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {node.name}
      </Typography>
      {hasChildren && (
        <KeyboardArrowRightIcon
          sx={{
            flexShrink: 0,
            fontSize: 20,
            transition: 'transform .15s',
            transform: expanded ? 'rotate(90deg)' : 'none',
            opacity: 0.6,
          }}
        />
      )}
    </Box>
  );
};

export default TopologyGraph;
