import { ButtonItem, PanelSection, PanelSectionRow, ProgressBarWithInfo, staticClasses } from '@decky/ui';
import { definePlugin, executeInTab, injectCssIntoTab, removeCssFromTab, toaster } from '@decky/api';
import { FaMedal } from 'react-icons/fa6';
import { NOTIFICATION_SSE_URL } from './shared/utils/fetcher';
import type { Notification } from '@/types/Notification';
import { getNotificationTab } from '@/shared/utils/utils';
import { ImgIcon } from '@/common/components/img-icon';

const sse = new EventSource(NOTIFICATION_SSE_URL);

const toasterClassName = `sentinel-toaster`;
const toasterContentClassName = `sentinel-toaster-content`;

// language=css
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

    /*override image container*/
    & div:has(img[data-name="ach"]) {
      width: 60px;
      height: 60px;
    }
  }

  .${toasterContentClassName} {
    width: 100%;
  }
`;

SteamClient.Apps.ScanForInstalledNonSteamApps();
//
// SteamClient.Apps.ScanForInstalledNonSteamApps();
// check if any of them is running
// it true, change view of QAM -sentinel to show list of achievements

//todo: start listening if game is running
sse.addEventListener('message', async (ev) => {
  const message: Notification = JSON.parse(ev.data);

  const duration = 7000;

  if (Object.keys(message).length > 0) {
    const notificationTab = (await getNotificationTab()) ?? '';
    //Note, injectCSSIntoTab returns a Promise
    const cssId = await injectCssIntoTab(notificationTab, toasterStyles);

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

    setTimeout(() => removeCssFromTab(notificationTab, cssId), duration);
  }
});

function Content() {
  return (
    <>
      <PanelSection title='Settings'>
        <PanelSectionRow>
          <ButtonItem layout='below' onClick={() => {}}>
            sssss
          </ButtonItem>
          <ProgressBarWithInfo nProgress={20} nTransitionSec={0.3} sTimeRemaining={'smm'} sOperationText={'ope'} />
        </PanelSectionRow>
      </PanelSection>
    </>
  );
}

export default definePlugin(() => {
  console.log('in define plugin');
  return {
    // The name shown in various decky menus
    name: 'Sentinel',
    // The element displayed at the top of your plugin's menu
    titleView: <div className={staticClasses.Title}>Sentinel</div>,
    // The content of your plugin's menu
    content: <Content />,
    // The icon displayed in the plugin list
    icon: <FaMedal />,
    // The function triggered when your plugin unloads
    onDismount() {
      console.log('unmounting');
    }
  };
});
