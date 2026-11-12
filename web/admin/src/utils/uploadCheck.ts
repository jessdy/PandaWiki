const IMAGE_EXTENSIONS = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'ico'];

export const MAX_IMAGE_SIZE = 10 * 1024 * 1024;
export const MAX_UPLOAD_SIZE = 100 * 1024 * 1024;

const extOf = (name: string) =>
  name.includes('.') ? name.split('.').pop()!.toLowerCase() : '';

/**
 * 按 accept 校验扩展名和大小，返回错误提示，通过时返回空串。
 * 只是提前拦截，后端会再按白名单和文件头复核。
 */
export const checkUploadFile = (file: File, accept?: string): string => {
  const ext = extOf(file.name);
  const rules = (accept || '')
    .split(',')
    .map(s => s.trim().toLowerCase())
    .filter(Boolean);
  const imageOnly =
    rules.length > 0 &&
    rules.every(
      r => r === 'image/*' || IMAGE_EXTENSIONS.includes(r.replace(/^\./, '')),
    );
  if (imageOnly) {
    if (!IMAGE_EXTENSIONS.includes(ext) || !file.type.startsWith('image/')) {
      return '仅支持 JPG、PNG、GIF、BMP、WEBP、ICO 格式的图片';
    }
    if (file.size > MAX_IMAGE_SIZE) {
      return '图片不能超过 10MB';
    }
    return '';
  }
  const exts = rules.filter(r => r.startsWith('.')).map(r => r.slice(1));
  if (exts.length > 0 && !exts.includes(ext)) {
    return `仅支持 ${exts.join('、')} 格式`;
  }
  if (file.size > MAX_UPLOAD_SIZE) {
    return '文件不能超过 100MB';
  }
  return '';
};
