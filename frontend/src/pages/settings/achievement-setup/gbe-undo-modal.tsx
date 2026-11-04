import '@/pages/settings/achievement-setup/achievement-setup.scss';
import { useEffect, useRef, useState, type FC } from 'react';
import { UndoGBESetup } from '@wa/sentinel/backend/generator/service';

type UndoState = 'confirm' | 'restoring' | 'succeeded' | 'failed';

export const GBEUndoModal: FC<{ isOpen: boolean; appId: string; gameName: string; onClose: () => void }> = ({
  isOpen,
  appId,
  gameName,
  onClose
}) => {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [state, setState] = useState<UndoState>('confirm');

  useEffect(() => {
    setState('confirm');
  }, [appId, gameName, isOpen]);

  useEffect(() => {
    if (isOpen && !dialogRef.current?.open) {
      dialogRef.current?.showModal();
    }
    if (!isOpen && dialogRef.current?.open) {
      dialogRef.current.close();
    }
  }, [isOpen]);

  const restoring = state === 'restoring';
  const finished = state === 'succeeded' || state === 'failed';

  const restore = async () => {
    if (state !== 'confirm') {
      return;
    }
    setState('restoring');
    try {
      await UndoGBESetup(appId);
      setState('succeeded');
    } catch {
      setState('failed');
    }
  };

  return (
    <dialog
      ref={dialogRef}
      className='app-dialog'
      aria-labelledby='gbe-undo-title'
      aria-describedby='gbe-undo-description'
      onCancel={(event) => {
        if (restoring) {
          event.preventDefault();
          return;
        }

        onClose();
      }}
    >
      <header className='app-dialog-header'>
        <h3 id='gbe-undo-title'>Undo achievement setup?</h3>
        <p id='gbe-undo-description'>Undo setup for {gameName || 'this game'}.</p>
      </header>
      <div className='app-dialog-content'>
        {state === 'confirm' && (
          <p>This will restore the game files backed up during achievement setup.</p>
        )}
        {state === 'restoring' && <p>Restoring Sentinel backups…</p>}
        {state === 'failed' && <p className='gbe-dialog-error'>Unable to undo achievement setup. Please try again.</p>}
        {state === 'succeeded' && <p>Achievement setup was undone.</p>}
      </div>
      <footer className='app-dialog-footer'>
        {state === 'confirm' && (
          <>
            <button className='outline' autoFocus onClick={onClose}>
              Cancel
            </button>
            <button data-variant='danger' onClick={() => void restore()}>
              Undo setup
            </button>
          </>
        )}
        {state === 'restoring' && <button data-variant='danger' disabled>Restoring…</button>}
        {finished && <button onClick={onClose}>Close</button>}
      </footer>
    </dialog>
  );
};
