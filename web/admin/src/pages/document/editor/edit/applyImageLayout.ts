import type { Editor } from '@tiptap/core';
import { Fragment, type Node as PMNode } from '@tiptap/pm/model';

const IMAGE_WIDTH = '25%';

function withImageWidth(attrs: Record<string, unknown>) {
  return { ...attrs, width: IMAGE_WIDTH };
}

function isImageNode(node: PMNode) {
  return node.type.name === 'image';
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

function blockRangePos(doc: PMNode, startIndex: number, endIndex: number) {
  let startPos = 0;
  for (let k = 0; k < startIndex; k++) startPos += doc.child(k).nodeSize;
  let endPos = startPos;
  for (let k = startIndex; k < endIndex; k++) endPos += doc.child(k).nodeSize;
  return { startPos, endPos };
}

/** 顶层连续的 block 级 image 节点（编辑器默认结构，非段落内）。 */
function collectTopLevelImageGroups(doc: PMNode): MergeGroup[] {
  const groups: MergeGroup[] = [];
  let i = 0;
  while (i < doc.childCount) {
    if (!isImageNode(doc.child(i))) {
      i++;
      continue;
    }
    const startIndex = i;
    const imageNodes: PMNode[] = [];
    while (i < doc.childCount && isImageNode(doc.child(i))) {
      imageNodes.push(doc.child(i));
      i++;
    }
    const { startPos, endPos } = blockRangePos(doc, startIndex, i);
    groups.push({ startPos, endPos, imageNodes });
  }
  return groups;
}

/** 连续的「仅含图片」段落。 */
function collectImageParagraphGroups(doc: PMNode): MergeGroup[] {
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
    const { startPos, endPos } = blockRangePos(doc, startIndex, i);
    groups.push({ startPos, endPos, imageNodes });
  }
  return groups;
}

function createImagesFragment(imageNodes: PMNode[]) {
  return Fragment.from(
    imageNodes.map(img =>
      img.type.create(withImageWidth(img.attrs as Record<string, unknown>)),
    ),
  );
}

/**
 * 将文档内所有图片统一为 25% 宽，并把连续图片收进同一段落以便横排。
 * 兼容 block 级 image（顶层）与 paragraph 内 inline image 两种结构。
 */
export function applyImageLayout(editor: Editor): boolean {
  const { state } = editor;
  const { schema } = state;
  const paragraph = schema.nodes.paragraph;
  if (!paragraph) return false;

  let hasImage = false;
  state.doc.descendants(node => {
    if (node.type.name === 'image') hasImage = true;
  });
  if (!hasImage) return false;

  let tr = state.tr;
  let changed = false;

  const applyMergeGroups = (doc: PMNode, groups: MergeGroup[]) => {
    for (const group of [...groups].sort((a, b) => b.startPos - a.startPos)) {
      tr = tr.replaceWith(
        group.startPos,
        group.endPos,
        paragraph.create(null, createImagesFragment(group.imageNodes)),
      );
      changed = true;
    }
  };

  // 先在当前 doc 上合并，再基于 tr.doc 做第二轮（避免位置偏移）。
  applyMergeGroups(state.doc, collectTopLevelImageGroups(state.doc));
  applyMergeGroups(tr.doc, collectImageParagraphGroups(tr.doc));

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

  if (!changed) return true;

  editor.view.dispatch(tr);
  return true;
}
