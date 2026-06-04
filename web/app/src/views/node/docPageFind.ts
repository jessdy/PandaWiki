export const DOC_FIND_MARK_CLASS = 'doc-page-find-mark';
export const DOC_FIND_ACTIVE_CLASS = 'doc-page-find-mark-active';

export function getDocFindRoot(): HTMLElement | null {
  return document.querySelector(
    '#doc-content .editor-container .tiptap.ProseMirror',
  );
}

export function clearDocFindHighlights(root?: HTMLElement | null) {
  const container = root ?? getDocFindRoot();
  if (!container) return;
  container.querySelectorAll(`mark.${DOC_FIND_MARK_CLASS}`).forEach(mark => {
    const parent = mark.parentNode;
    if (!parent) return;
    parent.replaceChild(document.createTextNode(mark.textContent || ''), mark);
    parent.normalize();
  });
}

export function highlightDocFindMatches(
  query: string,
  root?: HTMLElement | null,
): HTMLElement[] {
  const container = root ?? getDocFindRoot();
  if (!container) return [];

  clearDocFindHighlights(container);
  const term = query.trim();
  if (!term) return [];

  const lowerTerm = term.toLowerCase();
  const textNodes: Text[] = [];
  const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      const parent = node.parentElement;
      if (!parent) return NodeFilter.FILTER_REJECT;
      if (parent.closest(`mark.${DOC_FIND_MARK_CLASS}`)) {
        return NodeFilter.FILTER_REJECT;
      }
      if (!node.textContent) return NodeFilter.FILTER_REJECT;
      return NodeFilter.FILTER_ACCEPT;
    },
  });

  while (walker.nextNode()) {
    textNodes.push(walker.currentNode as Text);
  }

  const matches: HTMLElement[] = [];

  for (let i = textNodes.length - 1; i >= 0; i--) {
    const node = textNodes[i];
    const text = node.textContent || '';
    const lowerText = text.toLowerCase();
    let start = lowerText.indexOf(lowerTerm);
    if (start === -1) continue;

    const frag = document.createDocumentFragment();
    let cursor = 0;
    while (start !== -1) {
      if (start > cursor) {
        frag.appendChild(document.createTextNode(text.slice(cursor, start)));
      }
      const mark = document.createElement('mark');
      mark.className = DOC_FIND_MARK_CLASS;
      mark.textContent = text.slice(start, start + term.length);
      frag.appendChild(mark);
      matches.unshift(mark);
      cursor = start + term.length;
      start = lowerText.indexOf(lowerTerm, cursor);
    }
    if (cursor < text.length) {
      frag.appendChild(document.createTextNode(text.slice(cursor)));
    }
    node.parentNode?.replaceChild(frag, node);
  }

  return matches;
}

export function scrollToDocFindMatch(el: HTMLElement | undefined) {
  if (!el) return;
  const scrollContainer = document.querySelector('#scroll-container');
  if (!(scrollContainer instanceof HTMLElement)) {
    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    return;
  }

  const containerRect = scrollContainer.getBoundingClientRect();
  const elRect = el.getBoundingClientRect();
  const offset =
    scrollContainer.scrollTop +
    (elRect.top - containerRect.top) -
    containerRect.height / 2 +
    elRect.height / 2;

  scrollContainer.scrollTo({
    top: Math.max(0, offset),
    behavior: 'smooth',
  });
}
