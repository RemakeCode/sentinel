import './update-dialog.scss';
import { useEffect, useRef, useState, type FC } from 'react';
import { Events } from '@wailsio/runtime';
import { Download, Restart, State } from '@wa/sentinel/backend/updater/service';
import ReleaseNotes from '@/pages/settings/others/update/release-notes';

type Phase = 'loading' | 'idle' | 'downloading' | 'verifying' | 'installing' | 'ready' | 'error' | 'restarting';

const UpdateDialog: FC<{ isOpen: boolean; version: string; notes: string; onClose: () => void }> = ({ isOpen, version, notes, onClose }) => {
  const ref = useRef<HTMLDialogElement>(null);
  const [phase, setPhase] = useState<Phase>('loading');
  const [progress, setProgress] = useState<{ written: number; total: number } | null>(null);
  const [error, setError] = useState('');
  const busy = ['loading', 'downloading', 'verifying', 'installing', 'restarting'].includes(phase);

  const download = async () => {
    setPhase('downloading');
    setProgress(null);
    setError('');
    try {
      await Download();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setPhase('error');
    }
  };

  const open = async () => {
    setPhase('loading');
    setError('');
    try {
      if (await State() === 'ready') {
        setPhase('ready');
      } else {
        setPhase('idle');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setPhase('error');
    }
  };

  const restart = async () => {
    setPhase('restarting');
    setError('');
    try {
      await Restart();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setPhase('ready');
    }
  };

  useEffect(() => {
    const off = [
      Events.On('wails:updater:download-progress', (event: { data: { written: number; total: number } }) => setProgress(event.data)),
      Events.On('wails:updater:verifying', () => setPhase('verifying')),
      Events.On('wails:updater:installing', () => setPhase('installing')),
      Events.On('wails:updater:update-ready', () => setPhase('ready'))
    ];
    return () => off.forEach((unsubscribe) => unsubscribe());
  }, []);

  useEffect(() => {
    if (isOpen) {
      if (!ref.current?.open) ref.current?.showModal();
      void open();
    } else {
      ref.current?.close();
    }
  }, [isOpen]);

  const percent = progress && progress.total > 0
    ? Math.min(100, Math.max(0, Math.round(progress.written / progress.total * 100)))
    : undefined;
  const message = {
    loading: 'Preparing update…', idle: 'Review the changelog, then download when you are ready.', downloading: 'Downloading update…', verifying: 'Verifying update…',
    installing: 'Preparing update…', ready: 'Update ready. Restart when you are ready to apply it.',
    error: 'The update could not be completed.', restarting: 'Restarting Sentinel…'
  }[phase];

  return (
    <dialog ref={ref} className='app-dialog update-dialog' aria-labelledby='update-dialog-title'
      onCancel={(event) => { if (busy) event.preventDefault(); }} onClose={onClose}>
      <header className='app-dialog-header'>
        <h3 id='update-dialog-title'>Update Sentinel{version && ` to ${version}`}</h3>
      </header>
      <div className='app-dialog-content'>
        <p role='status'>{message}</p>
        {(phase === 'idle' || phase === 'loading') && <ReleaseNotes notes={notes} />}
        {phase === 'downloading' && (
          <div className='update-dialog-progress'>
            <progress max={100} value={percent} aria-label='Update download' />
            <span aria-hidden='true'>{percent === undefined ? 'Downloading…' : `${percent}%`}</span>
          </div>
        )}
        <div aria-live='polite'>
          {error && <p className='update-dialog-error'>{error}</p>}
        </div>
      </div>
      <footer className='app-dialog-footer'>
        <button type='button' className='outline' disabled={busy} onClick={onClose}>{phase === 'ready' ? 'Close' : 'Cancel'}</button>
        {(phase === 'idle' || phase === 'loading') && <button type='button' disabled={busy} onClick={() => void download()}>Download</button>}
        {phase === 'error' && <button type='button' onClick={() => void download()}>Retry Download</button>}
        {phase === 'ready' && <button type='button' onClick={() => void restart()}>Restart &amp; Apply</button>}
      </footer>
    </dialog>
  );
};

export default UpdateDialog;
