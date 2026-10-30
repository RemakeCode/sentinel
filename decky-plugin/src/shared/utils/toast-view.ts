import { executeInTab, injectCssIntoTab, removeCssFromTab } from '@decky/api';
import { findModule } from '@decky/ui';
import { sentinelLogger } from '@/shared/utils/logger';

export const toastViewSize = { width: 400, height: 110 };

const parentTab = 'Steam Big Picture Mode';
let removeToastViewStyle = () => Promise.resolve();

export function restoreToastView(): Promise<void> {
  const cleanup = removeToastViewStyle;
  removeToastViewStyle = () => Promise.resolve();
  return cleanup();
}

export function acquireToastView(element: HTMLElement | null): () => void {
  // Tray copies must not resize the shared notification view.
  if (!element?.closest('.DeckyGamepadUIPopupToast')) return () => {};

  const classes = findModule((module) => module?.GamepadToastPlaceholder) as
    | { GamepadToastPlaceholder: string }
    | undefined;
  if (!classes) {
    sentinelLogger.warn('Steam notification placeholder class is unavailable for resizing');
    return () => {};
  }

  const styleId = restoreToastView()
    // Let Steam observe the previous rule's removal before applying the same size again.
    .then(() =>
      executeInTab(
        parentTab,
        true,
        'new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve(true))))'
      )
    )
    .then(() =>
      injectCssIntoTab(
        parentTab,
        `.${classes.GamepadToastPlaceholder} {
        width: ${toastViewSize.width}px !important;
        height: ${toastViewSize.height}px !important;
      }`
      )
    )
    .catch((error): undefined => {
      sentinelLogger.warn('Unable to enlarge the Steam notification view', error);
      return undefined;
    });

  let removal: Promise<void> | undefined;
  const cleanup = () => {
    removal ??= styleId
      .then(async (id): Promise<void> => {
        if (id) await removeCssFromTab(parentTab, id);
      })
      .catch((error) => {
        sentinelLogger.warn('Unable to remove the Steam notification sizing rule', error);
      });
    return removal;
  };
  removeToastViewStyle = cleanup;
  return () => void cleanup();
}
