import DOMPurify from 'isomorphic-dompurify';

/** 用 DOMPurify 去掉脚本、事件属性和危险 URL，保留图片、公式和代码高亮。 */
export function sanitizeMarkdownHtml(html: string): string {
  return DOMPurify.sanitize(html, {
    ADD_TAGS: ['think', 'error'],
    ADD_ATTR: ['target'],
  });
}
