import { useEffect, useState } from 'react';
import { DialogButton, DialogControlsSection, DialogControlsSectionHeader, Field, ProgressBar } from '@decky/ui';
import { ReleaseNotes } from '@/pages/settings/updates/release-notes';
import {
  checkForUpdate,
  dispatchUpdateStatus,
  getUpdateStatus,
  prepareUpdate,
  subscribeUpdateStatus,
  type PreparedUpdate
} from '@/pages/settings/updates/update-status';

interface DeckyInstaller {
  call(route: string, ...args: unknown[]): Promise<unknown>;
}

//language=css
const styles = `
  .sentinel-update-progress {
    width: 100%;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .sentinel-update-progress-status {
    color: var(--gpColor-Blue, #1a9fff);
    font-size: 14px;
    font-weight: 700;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    text-align: right;
    overflow-wrap: anywhere;
  }
`;

async function requestDeckyInstall(update: PreparedUpdate): Promise<void> {
  const installer =
    typeof window === 'undefined' ? undefined : (window as unknown as { DeckyBackend?: DeckyInstaller }).DeckyBackend;
  if (!installer || typeof installer.call !== 'function') {
    throw new Error(
      'This Decky Loader version does not expose the installer. Update Decky or install the Sentinel ZIP through Decky settings.'
    );
  }
  if (!update.path.startsWith('/') || !/^[a-f0-9]{64}$/.test(update.sha256) || !update.version) {
    throw new Error('The verified update path, version, or checksum is invalid.');
  }
  await installer.call(
    'utilities/install_plugin',
    `file://${update.path}`,
    'Sentinel',
    update.version,
    update.sha256,
    2
  );
}

export function Updates() {
  const [state, setState] = useState(getUpdateStatus);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState('');
  const [installing, setInstalling] = useState(false);
  const preparing = state.phase === 'downloading' || state.phase === 'verifying';

  useEffect(() => subscribeUpdateStatus(setState), []);
  useEffect(() => {
    const current = getUpdateStatus();
    if (!current.release && current.phase !== 'downloading' && current.phase !== 'verifying') void check();
  }, []);

  const check = async () => {
    if (pending || preparing) return;
    setPending(true);
    setMessage('');
    try {
      const result = await checkForUpdate();
      if (!result.release) setMessage('No update available.');
    } catch (error) {
      setMessage(`Unable to check for updates: ${String(error)}`);
    } finally {
      setPending(false);
    }
  };

  const install = async () => {
    const release = state.release;
    if (!release || pending || installing || preparing) return;
    setInstalling(true);
    setMessage('');
    try {
      let result = getUpdateStatus();
      if (
        result.phase !== 'ready' ||
        !result.prepared ||
        result.prepared.version !== release.version ||
        result.prepared.sha256 !== release.sha256
      ) {
        dispatchUpdateStatus({
          ...result,
          release,
          phase: 'downloading',
          downloaded: 0,
          total: release.size,
          error: undefined
        });
        result = await prepareUpdate(release.version);
        dispatchUpdateStatus(result);
      }
      if (
        result.phase !== 'ready' ||
        !result.prepared ||
        result.prepared.version !== release.version ||
        result.prepared.sha256 !== release.sha256
      ) {
        throw new Error('The backend did not return a verified update. Check again and retry.');
      }
      await requestDeckyInstall(result.prepared);
      setMessage('Installation requested. Confirm in Decky’s prompt. The plugin reloads automatically.');
    } catch (error) {
      const current = getUpdateStatus();
      if (current.phase === 'downloading' || current.phase === 'verifying') {
        dispatchUpdateStatus({ ...current, release, phase: 'failed' });
      }
      setMessage(`Unable to update: ${String(error)}`);
    } finally {
      setInstalling(false);
    }
  };

  const busy = installing || preparing;
  const progress = state.total > 0 ? Math.min(100, Math.round((state.downloaded / state.total) * 100)) : undefined;
  let operation = 'Downloading update…';
  if (state.phase === 'verifying') operation = 'Verifying update…';
  else if (state.phase === 'ready') operation = 'Requesting installation…';
  let buttonLabel = 'Check';
  if (pending) buttonLabel = 'Checking…';
  else if (state.release) buttonLabel = 'Install Update';

  return (
    <DialogControlsSection>
      <style>{styles}</style>
      <DialogControlsSectionHeader>Updates</DialogControlsSectionHeader>
      <Field label='Updates' description={busy ? undefined : message || state.error} childrenContainerWidth='fixed'>
        {busy && (
          <div className='sentinel-update-progress'>
            <div className='sentinel-update-progress-status' role='status'>
              {operation}
            </div>
            <ProgressBar nProgress={progress} indeterminate={state.phase !== 'downloading' || progress === undefined} />
          </div>
        )}
        {!busy && (
          <DialogButton disabled={pending} onClick={state.release ? install : check}>
            {buttonLabel}
          </DialogButton>
        )}
      </Field>
      {state.release && <ReleaseNotes release={state.release} />}
    </DialogControlsSection>
  );
}
