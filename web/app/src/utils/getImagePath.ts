export const getImagePath = (path: string, basePath?: string) => {
  if (!path) return path;
  if (path.startsWith('http') || path.startsWith('blob')) {
    return path;
  }

  // 注意：不能用 `basePath || window._BASE_PATH_`——空字符串是合法 basePath，
  // 否则会落到尚未注入的 window._BASE_PATH_（undefined），拼成 "/undefined/..."。
  let prefix = '';
  if (typeof basePath === 'string') {
    prefix = basePath;
  } else if (
    typeof window !== 'undefined' &&
    typeof window._BASE_PATH_ === 'string'
  ) {
    prefix = window._BASE_PATH_;
  }
  if (prefix === 'undefined' || prefix === 'null') {
    prefix = '';
  }

  if (prefix && path.startsWith(prefix)) {
    return path;
  }
  return `${prefix}${path}`;
};
