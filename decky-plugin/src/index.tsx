import { DialogButton, Navigation } from '@decky/ui';
import { definePlugin, injectCssIntoTab, removeCssFromTab, routerHook, toaster } from '@decky/api';
import { FaBook, FaGear } from 'react-icons/fa6';
import { NOTIFICATION_SSE_URL } from './shared/utils/fetcher';
import type { Notification } from '@/shared/types/Notification';
import { getNotificationTab } from '@/shared/utils/utils';
import { ImgIcon } from '@/shared/components/img-icon';
import { ToastBody, ToastTitle } from '@/shared/components/toast';
import { initTracker } from '@/shared/utils/non-steam-game-tracker';
import MainPage from '@/pages/main';
import SettingsPage from '@/pages/settings';
import LibraryPage from '@/pages/library';
import AchievementsPage from '@/pages/achievements';
import { PiTrophy } from 'react-icons/pi';
import { playAudio } from '@/shared/utils/usePlayAudio';

let sse: EventSource | null = null;

initTracker().catch((error) => {
  console.error(error);
});

const toasterClassName = `sentinel-toaster`;
const toasterContentClassName = `sentinel-toaster-content`;

//language=css
const toasterStyles = `
  .${toasterClassName} {
    height: 55%;
    padding: 2px;
    border: 1px solid #3d4450;

    .${toasterContentClassName} {
      margin: 5px;
      width: 100%;
      height: 100%;
      display: flex;
      flex-direction: column;
      justify-content: center;
    }

    .sentinel-toast-title {
      font-size: 13px;
      font-weight: bold;
      text-overflow: ellipsis;
      overflow: hidden;
      white-space: nowrap;
    }

    .sentinel-toast-body {
      font-size: 12px;
      opacity: 0.7;
      text-overflow: ellipsis;
      overflow: hidden;
      white-space: nowrap;
    }

    .sentinel-toaster-progress-container {
      display: flex;
      gap: 2px;
      align-items: center;
    }

    .sentinel-toaster-progress-meta {
      font-weight: bold;
      font-size: small;
      color: var(--gpColor-Blue, #1a9fff);
    }

    & progress {
      width: 100%;
      height: 8px;
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

    img[data-name="ach"] {
      margin-left: 1px;
    }
  }

`;

let cssId: string | undefined;

async function connectSSE() {
  sse = new EventSource(NOTIFICATION_SSE_URL);

  const duration = 7000;

  sse.addEventListener('message', async (ev) => {
    const message: Notification = JSON?.parse(ev?.data);
    const notificationTab = (await getNotificationTab()) ?? '';

    cssId = cssId ? cssId : await injectCssIntoTab(notificationTab, toasterStyles);

    if (Object.keys(message).length > 0) {
      const showProgressToast = async () => {
        toaster.toast({
          title: <ToastTitle message={message} />,
          body: <ToastBody message={message} />,
          logo: <ImgIcon src={message.IconPath} />,
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
  });

  sse.addEventListener('open', async () => {
    console.log('Sentinel SSE is open for business');
  });
  sse.addEventListener('error', (error) => {
    console.log('Sentinel SSE error', error);
  });
}

export default definePlugin(() => {
  connectSSE().then(() => console.log('Connecting to SSE'));

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
      const notificationTab = await getNotificationTab();
      console.log('unmounting sentinel');
      if (cssId) {
        removeCssFromTab(notificationTab!, cssId);
      }
      if (sse) {
        sse.close();
      }
      routerHook.removeRoute('/sentinel/settings');
      routerHook.removeRoute('/sentinel/library');
      routerHook.removeRoute('/sentinel/games/:appId');
    }
  };
});
