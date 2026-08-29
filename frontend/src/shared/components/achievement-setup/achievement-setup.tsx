import './achievement-setup.scss';
import { useEffect, useState, type FC } from 'react';
import { Events } from '@wailsio/runtime';
import { type AchievementSetupSelection } from '@wa/sentinel/backend/generator/models';
import { GBESetupModal } from './gbe-setup-modal';
import { GBEUndoModal } from './gbe-undo-modal';

type OpenModal = 'setupModal' | 'undoModal' | null;

export const AchievementSetup: FC = () => {
  const [openModal, setOpenModal] = useState<OpenModal>(null);

  useEffect(() => {
    const unsubscribe = Events.On(
      'sentinel::achievement-setup-selected',
      (event: { data: AchievementSetupSelection }) => {
        setOpenModal(event.data.action === 'setup' ? 'setupModal' : 'undoModal');
      }
    );

    return unsubscribe;
  }, []);

  return (
    <>
      <GBESetupModal isOpen={openModal === 'setupModal'} onClose={() => setOpenModal(null)} />
      <GBEUndoModal isOpen={openModal === 'undoModal'} onClose={() => setOpenModal(null)} />
    </>
  );
};
