import { ButtonItem, PanelSection, PanelSectionRow, staticClasses } from '@decky/ui';
import { definePlugin, injectCssIntoTab, toaster } from '@decky/api';
import { FaMedal } from 'react-icons/fa6';
import { NOTIFICATION_SSE_URL } from '@/utils/fetcher';
import type { Notification } from '@/types/Notification';

const sse = new EventSource(NOTIFICATION_SSE_URL);

sse.addEventListener('message', (ev) => {
  const message: Notification = JSON.parse(ev.data);
  if (Object.keys(message).length > 0) {
    injectCssIntoTab('notificationtoasts_uid2', toasterStyles);
    toaster.toast({
      title: 'test',
      body: (
        <>
          <h2>can we div</h2> <br />
          dnddndn
        </>
      ),
      subtext: 'mysubtext',
      className: 'sentinel-toaster',
      logo: <ImgIcon src={message.IconPath} />
    });

    //TODO: set timeout and removeCSSfromTab after timeout
  }
});

const ImgIcon = ({ src }) => <img width='55px' src={src} />;

// language=css
const toasterStyles = `
  .sentinel-toaster {
    height: 100%;
    padding: 2px;
  }
`;

function Content() {
  return (
    <>
      <PanelSection title='Settings'>
        <PanelSectionRow>
          <ButtonItem layout='below' onClick={() => {}}>
            Roman
          </ButtonItem>
        </PanelSectionRow>
      </PanelSection>
    </>
  );
}

export default definePlugin(() => {
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
      // removeCssFromTab('notificationtoasts_uid2', toasterStyles);
    }
  };
});
