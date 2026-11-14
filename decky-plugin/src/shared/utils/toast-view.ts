import { injectCssIntoTab, removeCssFromTab } from '@decky/api';
import { findModule } from '@decky/ui';
import { sentinelLogger } from '@/shared/utils/logger';

export const toastViewSize = { width: 400, height: 110 };

const parentTab = 'Steam Big Picture Mode';
let stylePromise: Promise<string | undefined> | undefined;
let pendingToasts = 0;
let generation = 0;

export async function restoreToastView(): Promise<void> {
  const pendingStyle = stylePromise;
  stylePromise = undefined;
  pendingToasts = 0;
  generation++;
  try {
    const id = await pendingStyle;
    if (id) await removeCssFromTab(parentTab, id);
  } catch (error) {
    sentinelLogger.warn('Unable to remove the Steam notification sizing rule', error);
  }
}

// Reserve the view size before enqueueing, including while another toast is visible.
export async function prepareToastView(): Promise<() => void> {
  const currentGeneration = generation;
  pendingToasts++;
  if (!stylePromise) {
    const classes = findModule((module) => module?.GamepadToastPlaceholder) as
      | { GamepadToastPlaceholder: string }
      | undefined;
    stylePromise = classes
      ? Promise.resolve()
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
          })
      : Promise.resolve(undefined);
  }
  await stylePromise;

  let released = false;
  return () => {
    if (released || currentGeneration !== generation) return;
    released = true;
    pendingToasts--;
    if (pendingToasts === 0) void restoreToastView();
  };
}

export function acquireToastView(element: HTMLElement | null, releaseView: () => void): () => void {
  // Tray copies neither reveal popups nor release their size reservation.
  const popup = element?.closest('.DeckyGamepadUIPopupToast');
  const view = element?.ownerDocument.defaultView;
  if (!popup || !view) return () => {};

  let frame = 0;
  const revealWhenSized = () => {
    if (view.innerWidth >= toastViewSize.width && view.innerHeight >= toastViewSize.height) {
      popup.classList.add('sentinel-toast-ready');
    } else {
      frame = view.requestAnimationFrame(revealWhenSized);
    }
  };
  revealWhenSized();

  return () => {
    view.cancelAnimationFrame(frame);
    releaseView();
  };
}
