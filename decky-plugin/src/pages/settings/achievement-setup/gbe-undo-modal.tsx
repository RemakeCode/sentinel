import { type FC, useState } from 'react';
import { ConfirmModal, DialogBody, DialogBodyText } from '@decky/ui';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import { achievementSetupStyles } from '@/pages/settings/achievement-setup/achievement-setup-styles';

type UndoState = 'confirm' | 'restoring' | 'succeeded' | 'failed';

const fetcher = new Fetcher();

export const GBEUndoModal: FC<{ appId: string; gameName: string; closeModal: () => void }> = ({
  appId,
  gameName,
  closeModal
}) => {
  const [state, setState] = useState<UndoState>('confirm');

  const restoring = state === 'restoring';
  const finished = state === 'succeeded' || state === 'failed';

  const restore = async () => {
    if (state !== 'confirm') {
      return;
    }

    setState('restoring');
    try {
      await fetcher.post(`${BASE_URL}/gbe-setup/${appId}/undo`, {});
      setState('succeeded');
    } catch {
      setState('failed');
    }
  };

  return (
    <ConfirmModal
      strTitle='Undo Achievements Setup'
      strOKButtonText={state === 'confirm' ? 'Restore backups' : finished ? 'Close' : 'Restoring…'}
      strCancelButtonText={state === 'confirm' ? 'Back' : undefined}
      bOKDisabled={restoring}
      bDisableBackgroundDismiss={restoring}
      bHideCloseIcon={restoring}
      onOK={() => (finished ? closeModal() : void restore())}
      onCancel={() => {
        if (!restoring) {
          closeModal();
        }
      }}
      onEscKeypress={() => {
        if (!restoring) {
          closeModal();
        }
      }}
    >
      <style>{achievementSetupStyles}</style>
      <DialogBody className='sentinel-gbe-setup-undo-content'>
        {state === 'confirm' && (
          <DialogBodyText>
            Restore the matching DLL and steam_settings backups for <strong>{gameName || 'this game'}</strong>?
          </DialogBodyText>
        )}
        {state === 'restoring' && <DialogBodyText>Restoring Sentinel backups…</DialogBodyText>}
        {state === 'failed' && <DialogBodyText>Unable to undo achievement setup. Please try again.</DialogBodyText>}
        {state === 'succeeded' && <DialogBodyText>Achievement setup was undone.</DialogBodyText>}
      </DialogBody>
    </ConfirmModal>
  );
};
