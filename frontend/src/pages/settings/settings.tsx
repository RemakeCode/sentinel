import React, { useEffect, useState } from 'react';
import { Link } from 'react-router';
import { ArrowLeft, Eye, FolderOpen, Trash2, Volume2, VolumeOff } from 'lucide-react';

import { LoadConfig, AddEmulator, RemoveEmulator, ToggleEmulatorNotification } from '@wa/sentinel/backend/config/file';

import './settings.scss';
import EmptyState from '@/shared/components/EmptyState';
import { File, Emulator } from '@wa/sentinel/backend/config/models';

import { Dialogs } from '@wailsio/runtime';

declare global {
  interface Window {
    Oat: {
      toast: {
        show: (message: string, type?: 'success' | 'error' | 'info' | 'warning') => void;
      };
    };
  }
}

interface EmulatorItem {
  emu: Emulator;
  index: number;
}

const Settings: React.FC = () => {
  const [appConfig, setAppConfig] = useState<File | null>(null);
  const [darkMode, setDarkMode] = useState(true);

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      const cfg = await LoadConfig();
      setAppConfig(cfg);
    } catch (err) {
      console.error('Failed to load config:', err);
      window.Oat?.toast?.show('Failed to load settings', 'error');
    }
  };

  const handleAddEmulator = async () => {
    try {
      const selectedPath = await Dialogs.OpenFile({ CanChooseDirectories: true, CanChooseFiles: false, Title: 'Select A Folder to Watch' });

      if (selectedPath) {
        await AddEmulator(selectedPath);
        window.Oat?.toast?.show('Emulator path added', 'success');
        await loadConfig();
      }
    } catch (err) {
      console.error('Failed to add emulator:', err);
      window.Oat?.toast?.show('Failed to add emulator', 'error');
    }
  };

  const handleToggleNotify = async (index: number) => {
    try {
      await ToggleEmulatorNotification(index);
      await loadConfig();
    } catch (err) {
      console.error('Failed to toggle notification:', err);
      window.Oat?.toast?.show('Failed to update setting', 'error');
    }
  };

  const handleRemoveEmulator = async (index: number) => {
    try {
      await RemoveEmulator(index);
      window.Oat?.toast?.show('Emulator removed', 'success');
      await loadConfig();
    } catch (err) {
      console.error('Failed to remove emulator:', err);
      window.Oat?.toast?.show('Failed to remove emulator', 'error');
    }
  };

  const emulators = appConfig?.emulators || [];

  const allEmulators: EmulatorItem[] = emulators.map((emu: Emulator, index: number) => ({ emu, index }));

  return (
    <div className='settings-container'>
      <div className='settings-container-header'>
        <Link to='/' viewTransition>
          <ArrowLeft className='settings-back-icon' /> Back to Dashboard
        </Link>
      </div>
      <h2>Settings</h2>
      <div className='card'>
        <div className='flex  justify-between items-center'>
          <h4 className='settings-section-title'>
            <FolderOpen /> <span>Emulator Paths</span>
          </h4>
          <button data-variant='primary' onClick={handleAddEmulator}>
            <FolderOpen /> Add Emulator Folder
          </button>
        </div>
        <div className='divider' />

        <div className='settings-table'>
          {allEmulators.length === 0 ? (
            <EmptyState message='No emulator paths configured' />
          ) : (
            <table>
              <tbody>
                {allEmulators.map((record) => (
                  <tr key={record.index}>
                    <td width={'10%'}>
                      {record.emu.isDefault ? <span className='badge'>Default</span> : <span className='badge success'>Custom</span>}
                    </td>
                    <td width={'70%'}>
                      <code>{record.emu.path}</code>
                    </td>
                    <td>
                      <div className='settings-table-actions hstack gap-4'>
                        <label>
                          <input type='checkbox' role='switch' checked={record.emu.shouldNotify} onChange={() => handleToggleNotify(record.index)} />
                          {record.emu.shouldNotify ? <Volume2 width={20} fill='var(--success)' /> : <VolumeOff width={20} fill='var(--danger)' />}
                        </label>
                        |{<Trash2 onClick={() => handleRemoveEmulator(record.index)} className={record.emu.isDefault ? 'disabled' : ''} />}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      <hr className='settings-divider' />

      <div className='card settings-container-content-section-transparent-card'>
        <h4>
          <Eye /> Appearance
        </h4>
        <ul className='list horizontal'>
          <li>
            <div className='list-item-content'>
              <strong>Dark Mode</strong>
              <small>Use dark theme across the application</small>
            </div>
            <div className='list-item-actions'>
              <label className='switch'>
                <input type='checkbox' checked={darkMode} onChange={(e) => setDarkMode(e.target.checked)} />
                <span className='slider'></span>
              </label>
            </div>
          </li>
        </ul>
      </div>
    </div>
  );
};

export default Settings;
