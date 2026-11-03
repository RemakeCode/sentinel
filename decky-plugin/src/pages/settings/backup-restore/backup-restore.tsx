import { useCallback, useEffect, useState } from 'react';
import {
  ButtonItem,
  DialogBody,
  DialogButton,
  DialogControlsSection,
  DialogControlsSectionHeader,
  Field,
  Focusable
} from '@decky/ui';
import { FileSelectionType, openFilePicker, toaster } from '@decky/api';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import { showConfirmModal } from '@/shared/components/confirm';
import { sentinelLogger } from '@/shared/utils/logger';
import { getBackupStatus, subscribeBackupStatus } from '@/pages/settings/backup-restore/backup-status';

interface Snapshot {
  name: string;
  timestamp: number;
}

const fetcher = new Fetcher(BASE_URL);
const formatDate = (timestamp: number) =>
  new Date(timestamp * 1000)
    .toLocaleString('en-GB', {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      hour12: true
    })
    .replace(/\b(am|pm)\b/g, (period) => period.toUpperCase());

export function BackupRestoreContent() {
  const [directory, setDirectory] = useState('');
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [status, setStatus] = useState(getBackupStatus);
  const [pending, setPending] = useState(false);
  const [loading, setLoading] = useState(true);
  const running = pending || status?.status === 'running';

  const reportError = (error: unknown) => {
    sentinelLogger.error('Backup action failed', error);
    toaster.toast({ title: 'Backup & Restore', body: String(error) });
  };

  const refreshList = useCallback(async () => {
    setSnapshots((await fetcher.get<Snapshot[]>('/backups')) || []);
  }, []);

  useEffect(() => {
    let active = true;
    void Promise.all([fetcher.get<{ backupDirectory: string }>('/config'), fetcher.get<Snapshot[]>('/backups')])
      .then(([config, list]) => {
        if (!active) return;
        setDirectory(config.backupDirectory);
        setSnapshots(list || []);
      })
      .catch(reportError)
      .finally(() => {
        if (active) setLoading(false);
      });
    const unsubscribe = subscribeBackupStatus(setStatus);
    return () => {
      active = false;
      unsubscribe();
    };
  }, []);

  useEffect(() => {
    if (status?.operation === 'backup' && status.status === 'succeeded') void refreshList().catch(reportError);
  }, [status, refreshList]);

  const createBackup = async () => {
    setPending(true);
    try {
      await fetcher.post('/backups', {});
      await refreshList();
    } catch (error) {
      reportError(error);
    } finally {
      setPending(false);
    }
  };

  const restore = async (path: string) => {
    const confirmed = await showConfirmModal({
      title: 'Restore backup?',
      description: `${path.split('/').pop()}\n\nThis replaces your cached games, achievement progress, and images without merging.`,
      okText: 'Restore backup',
      cancelText: 'Cancel',
      destructive: true
    });
    if (!confirmed) return;
    setPending(true);
    try {
      await fetcher.post('/backups/restore', { path });
    } catch (error) {
      reportError(error);
    } finally {
      setPending(false);
    }
  };

  const restoreFromFile = async () => {
    try {
      const selected = await openFilePicker(
        FileSelectionType.FILE,
        directory || '/home/deck',
        true,
        false,
        undefined,
        undefined,
        false,
        true
      );
      if (selected.realpath) await restore(selected.realpath);
    } catch (error) {
      reportError(error);
    }
  };

  const changeDirectory = async () => {
    try {
      const selected = await openFilePicker(FileSelectionType.FOLDER, directory || '/home/deck', false, true);
      if (!selected.realpath) return;
      await fetcher.put('/config/backup-directory', { path: selected.realpath });
      setDirectory(selected.realpath);
      await refreshList();
    } catch (error) {
      reportError(error);
    }
  };

  const deleteBackup = async (name: string) => {
    if (
      !(await showConfirmModal({
        title: 'Delete backup?',
        description: `${name}\n\nThis backup will be permanently deleted.`,
        okText: 'Delete backup',
        destructive: true
      }))
    )
      return;
    try {
      await fetcher.delete(`/backups/${encodeURIComponent(name)}`);
      await refreshList();
    } catch (error) {
      reportError(error);
    }
  };

  return (
    <DialogBody>
      <DialogControlsSection>
        <DialogControlsSectionHeader>Backup & Restore</DialogControlsSectionHeader>
        <ButtonItem
          label='Backup folder'
          description={directory}
          disabled={loading || running}
          onClick={() => void changeDirectory()}
        >
          Change folder
        </ButtonItem>
        <ButtonItem
          label='Create backup'
          description='Back up your cached games, achievement progress, and images.'
          disabled={loading || running}
          onClick={() => void createBackup()}
        >
          Create backup
        </ButtonItem>
        <ButtonItem label='Restore from file' disabled={running || loading} onClick={() => void restoreFromFile()}>
          Choose file
        </ButtonItem>
        {running && <Field label={status?.operation === 'restore' ? 'Restoring backup…' : 'Creating backup…'} />}
      </DialogControlsSection>
      <DialogControlsSection>
        <DialogControlsSectionHeader>Available backups</DialogControlsSectionHeader>
        {loading ? (
          <Field label='Loading backups…' />
        ) : snapshots.length === 0 ? (
          <Field label='No backups in this folder' />
        ) : (
          snapshots.map((snapshot) => (
            <Field
              key={snapshot.name}
              label={snapshot.name}
              description={formatDate(snapshot.timestamp)}
              childrenLayout='inline'
              inlineWrap='keep-inline'
              verticalAlignment='center'
            >
              <Focusable flow-children='row' style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <DialogButton
                  style={{ minWidth: 0 }}
                  disabled={running}
                  onClick={() => void restore(`${directory.replace(/\/$/, '')}/${snapshot.name}`)}
                >
                  Restore
                </DialogButton>
                <DialogButton
                  style={{ minWidth: 0 }}
                  disabled={running}
                  onClick={() => void deleteBackup(snapshot.name)}
                >
                  Delete
                </DialogButton>
              </Focusable>
            </Field>
          ))
        )}
      </DialogControlsSection>
    </DialogBody>
  );
}
