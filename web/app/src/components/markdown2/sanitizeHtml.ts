const blockedTags =
  /<\/?(?:script|iframe|object|embed|link|meta|base|form)\b[^>]*>/gi;
const eventHandlers =
  /\s+on[a-z]+\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]+)/gi;
const dangerousUrls =
  /(\s(?:href|src|xlink:href)\s*=\s*)(['"])\s*(?:javascript:|data:text\/html)[^'"]*\2/gi;

/** 去掉 Markdown 渲染结果里的脚本和事件属性，保留图片、公式和代码高亮。 */
export function sanitizeMarkdownHtml(html: string): string {
  return html
    .replace(blockedTags, '')
    .replace(eventHandlers, '')
    .replace(dangerousUrls, '$1$2#$2');
}
