import { ButtonItem, Navigation, PanelSection, PanelSectionRow } from '@decky/ui';
import type { FC } from 'react';

const MainPage: FC = () => {
  return (
    <PanelSection title='Configuration'>
      <PanelSectionRow>
        <ButtonItem
          layout='below'
          onClick={() => {
            Navigation.Navigate('/sentinel/settings');
          }}
        >
          Open Settings
        </ButtonItem>
      </PanelSectionRow>
    </PanelSection>
  );
};

export default MainPage;
