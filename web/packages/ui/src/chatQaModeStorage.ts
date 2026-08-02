export type QaAppMode = 'training' | 'work';

/** 同页内切换模式时通知问答弹窗等订阅方更新样式 */
export const QA_APP_MODE_CHANGE_EVENT = 'panda-wiki:qa-app-mode';

export const CHAT_QA_MODE_STORAGE_KEY = 'panda_wiki_qa_app_mode';

/** demo 分支：固定实战模式，不允许切到培训 */
export const DEFAULT_QA_APP_MODE: QaAppMode = 'work';

export function parseValidQaAppMode(value: unknown): QaAppMode | null {
  const s = typeof value === 'string' ? value.trim() : '';
  if (s === 'training' || s === 'work') return s;
  return null;
}

export function getInitialQaAppMode(): QaAppMode {
  return 'work';
}

export function persistQaAppMode(_mode: QaAppMode): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem(CHAT_QA_MODE_STORAGE_KEY, 'work');
  window.dispatchEvent(
    new CustomEvent<QaAppMode>(QA_APP_MODE_CHANGE_EVENT, { detail: 'work' }),
  );
}
