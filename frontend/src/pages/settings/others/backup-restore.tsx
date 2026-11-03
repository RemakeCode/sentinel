import './backup-restore.scss';
import { useCallback, useEffect, useRef, useState, type FC } from 'react';
import { Dialogs } from '@wailsio/runtime';
import { Archive, ChevronRight, FileArchive, Folder, History, Plus, RotateCcw, Trash2 } from 'lucide-react';
import { motion, type Variants } from 'framer-motion';
import { Backup, Delete, List, Restore } from '@wa/sentinel/backend/backups/service';
import type { Snapshot } from '@wa/sentinel/backend/backups/models';
import { LoadConfig, SetBackupDirectory } from '@wa/sentinel/backend/config/file';
import { useBackup } from '@/shared/context/backup-context';
import { settingsSectionVariants } from '@/pages/settings/settings-motion';

const BackupRestore: FC = () => {
  const [directory, setDirectory] = useState('');
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [restoreFile, setRestoreFile] = useState<string | null>(null);
  const [deleteFile, setDeleteFile] = useState<string | null>(null);
  const dialogRef = useRef<HTMLDialogElement>(null);
  const deleteDialogRef = useRef<HTMLDialogElement>(null);
  const { status } = useBackup();
  const running = status?.status === 'running';

  const refreshList = useCallback(async () => {
    try {
      setSnapshots((await List()) || []);
    } catch (error) {
      console.error('Failed to list backups:', error);
      window.ot?.toast('Failed to load backups', 'Error', { variant: 'danger' });
    }
  }, []);

  useEffect(() => {
    void Promise.all([LoadConfig(), List()]).then(([config, list]) => {
      setDirectory(config?.backupDirectory || '');
      setSnapshots(list || []);
    }).catch((error) => {
      console.error('Failed to load backups:', error);
      window.ot?.toast('Failed to load backups', 'Error', { variant: 'danger' });
    });
  }, []);

  useEffect(() => {
    if (status?.operation === 'backup' && status.status === 'succeeded') void refreshList();
  }, [status, refreshList]);

  useEffect(() => {
    if (restoreFile && !dialogRef.current?.open) dialogRef.current?.showModal();
    if (!restoreFile && dialogRef.current?.open) dialogRef.current.close();
  }, [restoreFile]);

  useEffect(() => {
    if (deleteFile && !deleteDialogRef.current?.open) deleteDialogRef.current?.showModal();
    if (!deleteFile && deleteDialogRef.current?.open) deleteDialogRef.current.close();
  }, [deleteFile]);

  const chooseDirectory = async () => {
    try {
      const selected = await Dialogs.OpenFile({ CanChooseDirectories: true, CanChooseFiles: false, Title: 'Select backup folder' });
      if (!selected) return;
      await SetBackupDirectory(selected);
      setDirectory(selected);
      await refreshList();
    } catch (error) {
      console.error('Failed to change backup folder:', error);
      window.ot?.toast('Failed to change backup folder', 'Error', { variant: 'danger' });
    }
  };

  const restore = async (file: string) => {
    setRestoreFile(null);
    try { await Restore(file); } catch (error) {
      console.error('Restore failed:', error);
      if (String(error).includes('already running')) window.ot?.toast('A backup or restore is already running', 'Error', { variant: 'danger' });
    }
  };

  const importBackup = async () => {
    try {
      const selected = await Dialogs.OpenFile({ CanChooseFiles: true, CanChooseDirectories: false, Title: 'Select backup archive' });
      if (selected) setRestoreFile(selected);
    } catch (error) {
      console.error('Failed to select backup:', error);
      window.ot?.toast('Failed to select backup', 'Error', { variant: 'danger' });
    }
  };

  const remove = async (name: string) => {
    setDeleteFile(null);
    try {
      await Delete(name);
      await refreshList();
    } catch (error) {
      console.error('Failed to delete backup:', error);
      window.ot?.toast('Failed to delete backup', 'Error', { variant: 'danger' });
    }
  };

  return (
    <motion.div className='card settings-section settings-section-first backup-restore' variants={settingsSectionVariants as Variants}>
      <h4 className='settings-section-title'><Archive /> Backup &amp; Restore</h4>
      <hr className='divider' />
      <div className='backup-restore-intro'>
        <div><h4>Keep a backup of your data</h4><p>Back up your cached games, achievement progress, and images.</p></div>
        <div className='backup-restore-actions'>
        <button disabled={running} onClick={() => void Backup().catch((error) => {
          console.error('Backup failed:', error);
          if (String(error).includes('already running')) window.ot?.toast('A backup or restore is already running', 'Error', { variant: 'danger' });
        })}><Plus size={18} /> Create backup</button>
        <button className='outline' disabled={running} onClick={() => void importBackup()}><RotateCcw size={18} /> Restore from file…</button>
        </div>
      </div>
      <div className='backup-restore-folder'>
        <span className='backup-restore-icon'><Folder size={22} /></span>
        <div className='backup-restore-folder-details'><span>Backup folder</span><code className='backup-restore-directory'>{directory}</code></div>
        <button className='outline' onClick={() => void chooseDirectory()}>Change folder <ChevronRight size={18} /></button>
      </div>
      {running && <p aria-live='polite'>{status.operation === 'restore' ? 'Restoring backup…' : 'Creating backup…'}</p>}
      <h4 className='backup-restore-list-heading'>Available backups <span className='backup-restore-count'>{snapshots.length}</span></h4>
      {snapshots.length === 0 ? (
        <div className='backup-restore-empty'><span className='backup-restore-icon'><History size={22} /></span><div><strong>No backups in this folder</strong><p>Backups you create will appear here.</p></div></div>
      ) : (
        <ul className='backup-restore-list'>
          {snapshots.map((snapshot) => (
            <li className='backup-restore-item' key={snapshot.name}>
              <span className='backup-restore-icon'><FileArchive size={22} /></span>
              <div className='backup-restore-item-details'><strong>{snapshot.name}</strong><small>{new Date(snapshot.timestamp * 1000).toLocaleString('en-GB', { day: 'numeric', month: 'short', year: 'numeric', hour: 'numeric', minute: '2-digit', hour12: true }).replace(/\b(am|pm)\b/g, (period) => period.toUpperCase())}</small></div>
              <div className='backup-restore-actions'>
                <button className='outline' disabled={running} onClick={() => setRestoreFile(`${directory.replace(/\/$/, '')}/${snapshot.name}`)}><RotateCcw size={18} /> Restore</button>
                <button className='outline' disabled={running} onClick={() => setDeleteFile(snapshot.name)} aria-label={`Delete ${snapshot.name}`}><Trash2 size={18} /></button>
              </div>
            </li>
          ))}
        </ul>
      )}
      <dialog ref={dialogRef} className='app-dialog' aria-labelledby='restore-backup-title' onClose={() => setRestoreFile(null)}>
        <header className='app-dialog-header'>
          <h3 id='restore-backup-title'>Restore backup?</h3>
        </header>
        <div className='app-dialog-content'>
          <p><strong>{restoreFile?.split('/').pop()}</strong></p>
          <p>This will replace your cached games, achievement progress, and images. The existing cache will not be merged.</p>
        </div>
        <footer className='app-dialog-footer'>
          <button className='outline' onClick={() => setRestoreFile(null)}>Cancel</button>
          <button onClick={() => { if (restoreFile) void restore(restoreFile); }}>Restore backup</button>
        </footer>
      </dialog>
      <dialog ref={deleteDialogRef} className='app-dialog' aria-labelledby='delete-backup-title' onClose={() => setDeleteFile(null)}>
        <header className='app-dialog-header'>
          <h3 id='delete-backup-title'>Delete backup?</h3>
        </header>
        <div className='app-dialog-content'>
          <p><strong>{deleteFile}</strong></p>
          <p>This backup will be permanently deleted.</p>
        </div>
        <footer className='app-dialog-footer'>
          <button className='outline' onClick={() => setDeleteFile(null)}>Cancel</button>
          <button data-variant='danger' onClick={() => { if (deleteFile) void remove(deleteFile); }}>Delete backup</button>
        </footer>
      </dialog>
    </motion.div>
  );
};

export default BackupRestore;
