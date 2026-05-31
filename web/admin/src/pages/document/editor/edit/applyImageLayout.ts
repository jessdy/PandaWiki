import type { Editor } from '@tiptap/core';
import { Fragment, type Node as PMNode } from '@tiptap/pm/model';

const IMAGE_WIDTH = '25%';

function withImageWidth(attrs: Record<string, unknown>) {
  return { ...attrs, width: IMAGE_WIDTH };
}

function isImageOnlyParagraph(node: PMNode) {
  if (node.type.name !== 'paragraph' || node.childCount === 0) return false;
  for (let i = 0; i < node.childCount; i++) {
    if (node.child(i).type.name !== 'image') return false;
  }
  return true;
}

type MergeGroup = {
  startPos: number;
  endPos: number;
  imageNodes: PMNode[];
};

function collectMergeGroups(doc: PMNode): MergeGroup[] {
  const groups: MergeGroup[] = [];
  let i = 0;
  while (i < doc.childCount) {
    if (!isImageOnlyParagraph(doc.child(i))) {
      i++;
      continue;
    }
    const startIndex = i;
    const imageNodes: PMNode[] = [];
    while (i < doc.childCount && isImageOnlyParagraph(doc.child(i))) {
      const para = doc.child(i);
      for (let j = 0; j < para.childCount; j++) {
        imageNodes.push(para.child(j));
      }
      i++;
    }
    if (i - startIndex < 2) continue;

    let startPos = 0;
    for (let k = 0; k < startIndex; k++) startPos += doc.child(k).nodeSize;
    let endPos = startPos;
    for (let k = startIndex; k < i; k++) endPos += doc.child(k).nodeSize;
    groups.push({ startPos, endPos, imageNodes });
  }
  return groups;
}

/** 将文档内所有图片统一为 25% 宽，并把连续的单图段落合并为同一段落以便横排。 */
export function applyImageLayout(editor: Editor): boolean {
  const { state } = editor;
  const { schema, doc } = state;
  const paragraph = schema.nodes.paragraph;
  if (!paragraph) return false;

  const mergeGroups = collectMergeGroups(doc);
  let hasImage = false;
  doc.descendants(node => {
    if (node.type.name === 'image') hasImage = true;
  });
  if (!hasImage) return false;

  let tr = state.tr;
  let changed = false;

  for (const group of mergeGroups.sort((a, b) => b.startPos - a.startPos)) {
    const images = group.imageNodes.map(img =>
      img.type.create(withImageWidth(img.attrs as Record<string, unknown>)),
    );
    tr = tr.replaceWith(
      group.startPos,
      group.endPos,
      paragraph.create(null, Fragment.from(images)),
    );
    changed = true;
  }

  tr.doc.descendants((node, pos) => {
    if (node.type.name !== 'image') return true;
    if (node.attrs.width === IMAGE_WIDTH) return true;
    tr = tr.setNodeMarkup(
      pos,
      undefined,
      withImageWidth(node.attrs as Record<string, unknown>),
    );
    changed = true;
    return true;
  });

  if (!changed) {
    // 已全部是 25% 且无段落可合并，仍视为成功（用户重复点击）
    return true;
  }

  editor.view.dispatch(tr);
  return true;
}
