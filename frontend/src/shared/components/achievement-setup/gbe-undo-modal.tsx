import './achievement-setup.scss';
import { useEffect, useRef, useState, type FC } from 'react';
import { Events } from '@wailsio/runtime';
import { UndoGBESetup } from '@wa/sentinel/backend/generator/service';
import { type AchievementSetupSelection } from '@wa/sentinel/backend/generator/models';

type UndoState = 'confirm' | 'restoring' | 'succeeded' | 'failed';

export const GBEUndoModal: FC<{ isOpen: boolean; onClose: () => void }> = ({ isOpen, onClose }) => {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [request, setRequest] = useState<AchievementSetupSelection>();
  const [state, setState] = useState<UndoState>('confirm');

  useEffect(() => {
    const unsubscribe = Events.On(
      'sentinel::achievement-setup-selected',
      (event: { data: AchievementSetupSelection }) => {
        if (event.data.action !== 'undo') {
          return;
        }

        setRequest(event.data);
        setState('confirm');
      }
    );

    return unsubscribe;
  }, []);

  useEffect(() => {
    if (isOpen && request && !dialogRef.current?.open) {
      dialogRef.current?.showModal();
    }
    if (!isOpen && dialogRef.current?.open) {
      dialogRef.current.close();
    }
  }, [isOpen, request]);

  if (!request) {
    return null;
  }

  const restoring = state === 'restoring';
  const finished = state === 'succeeded' || state === 'failed';

  const restore = async () => {
    if (state !== 'confirm') {
      return;
    }
    setState('restoring');
    try {
      await UndoGBESetup(request.appId);
      setState('succeeded');
    } catch {
      setState('failed');
    }
  };

  return (
    <dialog
      ref={dialogRef}
      className='gbe-setup-dialog'
      onCancel={(event) => {
        if (restoring) {
          event.preventDefault();
          return;
        }

        onClose();
      }}
    >
      <header className='gbe-setup-dialog-header gbe-setup-dialog-header--without-progress'>
        <h3>Undo Achievement Setup</h3>
      </header>
      <div className='gbe-setup-dialog-content'>
        {state === 'confirm' && (
          <p>
            Restore the matching DLL and steam_settings backups for <strong>{request.gameName || 'this game'}</strong>?
          </p>
        )}
        {state === 'restoring' && <p>Restoring Sentinel backups…</p>}
        {state === 'failed' && (
          <p className='gbe-setup-dialog-error'>Unable to undo achievement setup. Please try again.</p>
        )}
        {state === 'succeeded' && <p>Achievement setup was undone.</p>}
      </div>
      <footer>
        {state === 'confirm' && (
          <>
            <button onClick={() => void restore()}>Restore backups</button>
            <button className='outline' onClick={onClose}>
              Back
            </button>
          </>
        )}
        {state === 'restoring' && <button disabled>Restoring…</button>}
        {finished && <button onClick={onClose}>Close</button>}
      </footer>
    </dialog>
  );
};
