import { staticClasses } from '@decky/ui';
import { definePlugin, executeInTab, injectCssIntoTab, removeCssFromTab, routerHook, toaster } from '@decky/api';
import { FaMedal } from 'react-icons/fa6';
import { NOTIFICATION_SSE_URL } from './shared/utils/fetcher';
import type { Notification } from '@/shared/types/Notification';
import { getNotificationTab } from '@/shared/utils/utils';
import { ImgIcon } from '@/shared/components/img-icon';
//import { initNonSteamGameTracker } from '@/shared/utils/non-steam-game-tracker';
import MainPage from '@/pages/main';
import SettingsPage from '@/pages/settings';

let sse: EventSource | null = null;

//initNonSteamGameTracker();

const toasterClassName = `sentinel-toaster`;
const toasterContentClassName = `sentinel-toaster-content`;

const toasterStyles = `
  .${toasterClassName} {
    height: 90%;
    padding: 2px;
    border: 1px solid #3d4450;

    .sentinel-toaster-progress-container {
      display: flex;
      gap: 2px;
      align-items: center;
      margin-top: 4px;
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

    & div:has(img[data-name="ach"]) {
      width: 60px;
      height: 60px;
    }
  }

  .${toasterContentClassName} {
    width: 100%;
  }
`;
// const fetcher = new Fetcher();

let cssId: string | undefined;

async function connectSSE() {
  sse = new EventSource(NOTIFICATION_SSE_URL);

  const duration = 7000;

  sse.addEventListener('message', async (ev) => {
    const message: Notification = JSON?.parse(ev?.data);
    const notificationTab = (await getNotificationTab()) ?? '';

    cssId = cssId ? cssId : await injectCssIntoTab(notificationTab, toasterStyles);

    console.log({ cssId });

    if (Object.keys(message).length > 0) {
      const showProgressToast = async () => {
        toaster.toast({
          title: message.Title,
          body: message.Message,
          logo: <ImgIcon src={message.IconPath} />,
          playSound: false,
          eType: 3,
          expiration: 0,
          className: toasterClassName,
          contentClassName: toasterContentClassName,
          duration
        });

        if (message.IsProgress) {
          const value = (message.Progress / message.MaxProgress) * 100;

          //language=javascript
          const progressEl = `
            (function() {
              const toastEl = document.querySelector(' .${toasterContentClassName}');

              if (toastEl) {
                const progressContainer = document.createElement('div');
                progressContainer.className = 'sentinel-toaster-progress-container';

                const progressBar = document.createElement('progress');
                progressBar.value = '${value}';
                progressBar.max = 100;

                const progressMeta = document.createElement('div');
                progressMeta.className = 'sentinel-toaster-progress-meta';
                progressMeta.textContent = '${message.Progress}/${message.MaxProgress}' 
                
                progressContainer.append(...[progressBar, progressMeta])
                toastEl.appendChild(progressContainer);
              }
            })();
        `;
          await executeInTab(notificationTab, false, progressEl);
        }
      };

      await showProgressToast();

      //setTimeout(() => removeCssFromTab(notificationTab, cssId), duration + 500);
    }
  });

  sse.addEventListener('error', (error) => {
    console.log('Sentinel SSE error', error);
  });
}

export default definePlugin(() => {
  connectSSE().then(() => console.log('Connecting to SSE'));

  routerHook.addRoute('/sentinel/settings', () => <SettingsPage />);

  return {
    name: 'Sentinel',
    titleView: <div className={staticClasses.Title}>Sentinel</div>,
    content: <MainPage />,
    icon: <FaMedal />,
    async onDismount() {
      const notificationTab = await getNotificationTab();
      console.log('unmounting sentinel');
      removeCssFromTab(notificationTab!, cssId!);
      if (sse) sse.close();
      routerHook.removeRoute('/sentinel/settings');
    }
  };
});
