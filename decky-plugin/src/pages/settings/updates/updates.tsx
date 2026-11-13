import { useEffect, useState } from 'react';
import { DialogButton, DialogControlsSection, DialogControlsSectionHeader, Field, Focusable } from '@decky/ui';
import { checkForUpdate, getUpdateStatus, prepareUpdate, subscribeUpdateStatus, type PreparedUpdate } from '@/pages/settings/updates/update-status';

interface DeckyInstaller {
  call(route: string, ...args: unknown[]): Promise<unknown>;
}

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
  // Internal Loader interface, isolated here for compatibility with future versions.
  await installer.call(
    'utilities/install_plugin',
    `file://${update.path}`,
    'Sentinel',
    update.version,
    update.sha256,
    2
  );
}

const styles = `
  .sentinel-release-notes {
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    max-height: 260px;
    overflow-y: auto;
    padding: 8px;
  }
`;

export function Updates() {
  const [state, setState] = useState(getUpdateStatus);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState('');
  const preparing = state.phase === 'downloading' || state.phase === 'verifying';
  const busy = pending || preparing;

  useEffect(() => {
    const current = getUpdateStatus();
    if (current.phase !== 'downloading' && current.phase !== 'verifying') void check();
  }, []);

  useEffect(() => {
    if (!preparing) return undefined;
    return subscribeUpdateStatus(setState);
  }, [preparing]);

  const check = async () => {
    const current = getUpdateStatus();
    if (pending || current.phase === 'downloading' || current.phase === 'verifying') return;
    setPending(true);
    setMessage('');
    try {
      const result = await checkForUpdate();
      setState(result);
      if (!result.release) setMessage('No update available.');
    } catch (error) {
      setMessage(`Unable to check for updates: ${String(error)}`);
    } finally {
      setPending(false);
    }
  };

  const update = async () => {
    const release = state.release;
    if (busy || !release) return;
    setPending(true);
    setMessage('Preparing update…');
    setState({ ...state, phase: 'downloading', downloaded: 0, total: release.size, error: undefined });
    try {
      const result = await prepareUpdate(release.version);
      setState(result);
      if (result.phase !== 'ready' || !result.prepared || result.prepared.version !== release.version || result.prepared.sha256 !== release.sha256) {
        throw new Error('The backend did not return a verified update. Check again and retry.');
      }
      await requestDeckyInstall(result.prepared);
      setMessage('Installation requested. Confirm in Decky’s prompt. The plugin reloads automatically.');
    } catch (error) {
      setState({ ...getUpdateStatus(), phase: 'failed' });
      setMessage(`Unable to update: ${String(error)}`);
    } finally {
      setPending(false);
    }
  };
  const progress = state.total > 0 ? Math.min(100, Math.round((state.downloaded / state.total) * 100)) : 0;

  return (
    <DialogControlsSection>
      <style>{styles}</style>
      <DialogControlsSectionHeader>Updates{state.release ? ' • Available' : ''}</DialogControlsSectionHeader>
      <Field
        label='Check for updates'
        description={message || state.error || (state.phase === 'checking' ? 'Checking…' : '')}
      >
        <DialogButton disabled={busy} onClick={check}>
          Check
        </DialogButton>
      </Field>
      {state.release && (
        <>
          <Field label='Shared release notes' description='This release may include both desktop and Decky changes.' />
          <Focusable className='sentinel-release-notes'>
            {state.release.notes || 'No release notes provided.'}
          </Focusable>
          <Field
            label={
              busy && preparing
                ? state.phase === 'verifying'
                  ? 'Verifying update…'
                  : `Downloading update… ${progress}%`
                : 'Update available'
            }
          >
            <DialogButton disabled={busy} onClick={update}>
              {state.phase === 'ready' ? 'Install update' : 'Update'}
            </DialogButton>
          </Field>
        </>
      )}
    </DialogControlsSection>
  );
}
