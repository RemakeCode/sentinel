import { ButtonItem, PanelSection, PanelSectionRow, staticClasses } from '@decky/ui';
import { addEventListener, definePlugin, removeEventListener, toaster } from '@decky/api';
import { useEffect, useState } from 'react';
import { FaShip } from 'react-icons/fa';

// import logo from "../assets/logo.png";

function Content() {
  const [result, setResult] = useState<number | undefined>();

  const onClick = async () => {
    const val = Math.random();
    setResult(val)
  };

  return (
    <PanelSection title='Panel Section'>
      <PanelSectionRow>
        <ButtonItem layout='below' onClick={onClick}>
         Sum
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>

      </PanelSectionRow>

      {/* <PanelSectionRow>
        <div style={{ display: "flex", justifyContent: "center" }}>
          <img src={logo} />
        </div>
      </PanelSectionRow> */}

      {/*<PanelSectionRow>
        <ButtonItem
          layout="below"
          onClick={() => {
            Navigation.Navigate("/decky-plugin-test");
            Navigation.CloseSideMenus();
          }}
        >
          Router
        </ButtonItem>
      </PanelSectionRow>*/}
    </PanelSection>
  );
}

export default definePlugin(() => {
  console.log('Template plugin initializing, this is called once on frontend startup');

  return {
    // The name shown in various decky menus
    name: 'Test Plugin',
    // The element displayed at the top of your plugin's menu
    titleView: <div className={staticClasses.Title}>Sentinel</div>,
    // The content of your plugin's menu
    content: <Content />,
    // The icon displayed in the plugin list
    icon: <FaShip />,
    // The function triggered when your plugin unloads
    onDismount() {
      console.log('Unloading');
    }
  };
});
