import { DialogButton, Navigation } from '@decky/ui';
import { definePlugin, injectCssIntoTab, removeCssFromTab, routerHook, toaster } from '@decky/api';
import { FaBook, FaGear } from 'react-icons/fa6';
import { NOTIFICATION_SSE_URL } from '@/shared/utils/fetcher';
import type { Notification } from '@/shared/types/Notification';
import { getNotificationTab } from '@/shared/utils/utils';
import { steamToastClasses, ToastBody, ToastTitle } from '@/shared/components/toast';
import { initTracker, type TrackerCleanup } from '@/shared/utils/non-steam-game-tracker';
import MainPage from '@/pages/main';
import SettingsPage from '@/pages/settings/settings';
import LibraryPage from '@/pages/library';
import AchievementsPage from '@/pages/achievements';
import { PiTrophy } from 'react-icons/pi';
import { playAudio } from '@/shared/utils/usePlayAudio';
import { SSEController } from '@/shared/utils/sse-controller';
import { sentinelLogger } from '@/shared/utils/logger';
import { rareAchievementGlowStyles } from '@/shared/rare-achievement-glow';
import { dispatchGBESetupUpdate } from '@/shared/utils/gbe-setup-events';
import { dispatchLibraryUpdate } from '@/shared/utils/library-update-events';
import type { Update } from '@/shared/types/_generated/sentinel/backend/generator/models';
import { restoreToastView, toastViewSize } from '@/shared/utils/toast-view';

let sseController: SSEController | null = null;

const toasterClassName = `sentinel-toaster`;
const toasterContentClassName = `sentinel-toaster-content`;

//language=css
const toasterStyles = `
  ${rareAchievementGlowStyles}

  .sentinel-rare-achievement-glow--toast {
    --rare-achievement-glow-radius: 6px;
  }

  .sentinel-rare-achievement-glow--toast > img[data-name="ach"] {
    margin-left: 0;
  }

  .${toasterClassName} {
    /* Steam supplies the card background, shadow, and shape. */
    width: ${toastViewSize.width}px;
    height: ${toastViewSize.height}px;
    box-sizing: border-box;
    padding: 3px 6px 4px;

    .${toasterContentClassName} {
      margin: 0;
      width: 100%;
      height: 100%;
      min-width: 0;
      display: grid;
      grid-template-columns: minmax(0, 1fr);
      grid-template-rows: auto minmax(0, 1fr);
      gap: 6px;
      padding: 4px;
      box-sizing: border-box;


      /* Release the native slots while retaining Steam's card and text styling. */
      > .${steamToastClasses?.Header},
      > .${steamToastClasses?.Body} {
        height: auto;
        min-height: 0;
        min-width: 0;
        white-space: normal;
        overflow: visible;
        display: block;
        -webkit-line-clamp: unset;
      }

      > .${steamToastClasses?.Header} > .${steamToastClasses?.Title} {
        display: block;
        width: 100%;
        margin-right: 0;
      }
    }

    .sentinel-toast-game-name {
      text-overflow: ellipsis;
      overflow: hidden;
      white-space: nowrap;
      line-height: 12px;
      border-bottom: 1px solid #3d4450;
      padding-block-end: 2px;
    }

    .sentinel-toast-layout {
      display: flex;
      align-items: center;
      gap: 12px;
      min-width: 0;
      height: 100%;
    }

    .sentinel-toast-icon {
      flex: 0 0 auto;
      height: 100%;
      aspect-ratio: 1;
    }

    .sentinel-toast-icon .sentinel-img-icon {
      display: block;
      width: 100%;
      height: 100%;
    }

    .sentinel-toast-details {
      display: flex;
      flex: 1 1 auto;
      flex-direction: column;
      gap: 4px;
      min-width: 0;
    }

    .sentinel-toast-details--progress {
      height: 100%;
      gap: 2px;
    }

    .sentinel-toast-title {
      text-overflow: ellipsis;
      overflow: hidden;
      white-space: nowrap;
      font-size: 14px;
      line-height: 18px;
    }

    .sentinel-toast-body {
      display: -webkit-box;
      height: auto;
      max-height: 32px;
      overflow: hidden;
      white-space: normal;
      overflow-wrap: anywhere;
      text-overflow: ellipsis;
      -webkit-box-orient: vertical;
      -webkit-line-clamp: 2;
      font-size: 12px;
      line-height: 14px;
    }

    .sentinel-toaster-progress-container {
      display: flex;
      flex-direction: column;
      gap: 2px;
      margin-top: auto;
      font-weight: bold;
      margin-block-end: 1px;
    }

    .sentinel-toaster-progress-meta {
      color: var(--gpColor-Blue, #1a9fff);
      align-self: flex-end;
      font-size: 10px;
      line-height: 10px;
    }

    & progress {
      width: 100%;
      height: 8px;
      flex-shrink: 0;
      appearance: none;
      border: 0;
      background: #3d4450;
      border-radius: 10px;

      &::-webkit-progress-bar {
        background: #3d4450;
        border-radius: 10px;
      }

      &::-webkit-progress-value {
        background: var(--gpColor-Blue, #1a9fff);
        border-radius: 10px;
        transition: width 200ms cubic-bezier(0.4, 0, 0.2, 1);
      }
    }
  }

`;

let cssId: string | undefined;

const duration = 7000;

async function handleNotificationMessage(ev: MessageEvent<string>) {
  let envelope: { messageType?: string; payload?: unknown };
  try {
    envelope = JSON.parse(ev.data);
  } catch {
    sentinelLogger.warn('Ignoring malformed Sentinel SSE message');
    return;
  }
  if (envelope.messageType === 'dataUpdated') {
    dispatchLibraryUpdate();
    return;
  }
  if (envelope.messageType === 'gbeSetup') {
    dispatchGBESetupUpdate(envelope.payload as Update);
    return;
  }
  if (envelope.messageType === 'achievement') {
    const message = envelope.payload as Notification;
    const notificationTab = (await getNotificationTab()) ?? '';
    cssId = cssId ? cssId : await injectCssIntoTab(notificationTab, toasterStyles);

    if (Object.keys(message).length > 0) {
      const showProgressToast = async () => {
        toaster.toast({
          title: <ToastTitle message={message} />,
          body: <ToastBody message={message} />,
          playSound: false,
          eType: 3,
          expiration: 0,
          className: toasterClassName,
          contentClassName: toasterContentClassName,
          duration
        });

        if (message.SoundFile) {
          await playAudio(message.SoundFile);
        }
      };
      await showProgressToast();
    }
    return;
  }

  sentinelLogger.warn(`Ignoring unknown Sentinel SSE message type: ${String(envelope.messageType)}`);
}

export default definePlugin(() => {
  sseController = new SSEController({ url: NOTIFICATION_SSE_URL, onMessage: handleNotificationMessage });
  sseController.start();
  const cleanupTracker: TrackerCleanup = initTracker();

  routerHook.addRoute('/sentinel/settings', () => <SettingsPage />);
  routerHook.addRoute('/sentinel/library', () => <LibraryPage />);
  routerHook.addRoute('/sentinel/games/:appId', () => <AchievementsPage />);

  return {
    name: 'Sentinel',
    titleView: (
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
        <span>Sentinel</span>
        <div style={{ display: 'flex', gap: '8px' }}>
          <DialogButton
            onClick={() => Navigation.Navigate('/sentinel/library')}
            style={{ padding: '8px', minWidth: 'fit-content' }}
          >
            <FaBook />
          </DialogButton>
          <DialogButton
            onClick={() => Navigation.Navigate('/sentinel/settings')}
            style={{ padding: '8px', minWidth: 'fit-content' }}
          >
            <FaGear />
          </DialogButton>
        </div>
      </div>
    ),
    content: <MainPage />,
    icon: <PiTrophy />,
    async onDismount() {
      sseController?.dispose();
      sseController = null;
      await restoreToastView();
      const notificationTab = await getNotificationTab();
      sentinelLogger.log('unmounting sentinel');
      if (cssId) {
        removeCssFromTab(notificationTab!, cssId);
      }
      cleanupTracker();
      routerHook.removeRoute('/sentinel/settings');
      routerHook.removeRoute('/sentinel/library');
      routerHook.removeRoute('/sentinel/games/:appId');
    }
  };
});
