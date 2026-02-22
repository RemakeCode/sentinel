import React, { useEffect, useState } from 'react';
import { Link } from 'react-router';
import { ArrowLeft, DatabaseSearchIcon, Eye, FolderOpen, Locate, Trash2, Volume2, VolumeOff } from 'lucide-react';

import {
  LoadConfig,
  AddEmulator,
  RemoveEmulator,
  ToggleEmulatorNotification,
  SetSteamAPIKey,
  GetSteamAPIKeyMasked,
  GetSteamDataSource,
  SetSteamDataSource
} from '@wa/sentinel/backend/config/file';

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
  const [steamAPIKey, setSteamAPIKey] = useState('');
  const [maskedSteamAPIKey, setMaskedSteamAPIKey] = useState('');
  const [steamAPIKeyHasError, setSteamAPIKeyHasError] = useState<boolean>(false);
  const [steamDataSource, setSteamDataSource] = useState<string>('external-source');

  useEffect(() => {
    loadConfig();
    loadSteamAPIKey();
    loadSteamDataSource();
  }, []);

  const loadSteamAPIKey = async () => {
    try {
      const maskedKey = await GetSteamAPIKeyMasked();
      setMaskedSteamAPIKey(maskedKey);
    } catch (err) {
      console.error('Failed to load Steam API key:', err);
      window.Oat?.toast?.show('Failed to load Steam API key', 'error');
    }
  };

  const loadSteamDataSource = async () => {
    try {
      const dataSource = await GetSteamDataSource();
      setSteamDataSource(dataSource);
    } catch (err) {
      console.error('Failed to load Steam data source:', err);
      window.Oat?.toast?.show('Failed to load Steam data source', 'error');
    }
  };

  const handleSteamDataSourceChange = async (value: string) => {
    try {
      await SetSteamDataSource(value);
      setSteamDataSource(value);
      window.Oat?.toast?.show('Steam data source updated', 'success');
    } catch (err) {
      console.error('Failed to save Steam data source:', err);
      window.Oat?.toast?.show('Failed to save Steam data source', 'error');
    }
  };

  let timeout: ReturnType<typeof setTimeout>;
  const handleSaveSteamAPIKey = async () => {
    if (timeout) {
      clearTimeout(timeout);
    }

    try {
      if (steamAPIKey === '') {
        setSteamAPIKeyHasError(true);
        timeout = setTimeout(() => setSteamAPIKeyHasError(false), 5000);
        return;
      }
      await SetSteamAPIKey(steamAPIKey);

      window.Oat?.toast?.show('Steam API key saved', 'success');

      //Reload from backend
      await loadSteamAPIKey();
      //Clear input field
      setSteamAPIKey('');
    } catch (err) {
      console.error('Failed to save Steam API key:', err);
      window.Oat?.toast?.show('Failed to save Steam API key', 'error');
    }
  };

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
    <>
      <div className='settings-container'>
        <div className='settings-container-header'>
          <Link to='/' viewTransition>
            <ArrowLeft className='settings-back-icon' /> Back to Dashboard
          </Link>
        </div>
        <h2>Settings</h2>
        <div className='card settings-section'>
          <div className='flex justify-between items-center'>
            <h4 className='settings-section-title'>
              <FolderOpen /> <span>Emulator Paths</span>
            </h4>
            <button data-variant='primary' onClick={handleAddEmulator}>
              <FolderOpen /> Add Emulator Folder
            </button>
          </div>
          <hr className='divider' />
          <div className='settings-table'>
            {allEmulators.length === 0 ? (
              <EmptyState message='No emulator paths configured' />
            ) : (
              <table>
                <tbody>
                  {allEmulators.map((record) => (
                    <tr key={record.index}>
                      <td className='settings-table-cell-type'>
                        {record.emu.isDefault ? <span className='badge'>Default</span> : <span className='badge success'>Custom</span>}
                      </td>
                      <td className='settings-table-cell-path'>
                        <code>{record.emu.path}</code>
                      </td>
                      <td className='settings-table-cell-actions'>
                        <div className='settings-table-actions hstack gap-4'>
                          <label>
                            <input
                              type='checkbox'
                              role='switch'
                              checked={record.emu.shouldNotify}
                              onChange={() => handleToggleNotify(record.index)}
                            />
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
          <hr className='divider' />
          <div className='settings-table-steam-datasource'>
            <label>Data Source</label>
            <div className='radio-group'>
              <label className='radio-option'>
                <input
                  type='radio'
                  name='steamDataSource'
                  value='steam-key'
                  checked={steamDataSource === 'steam-key'}
                  onChange={(e) => handleSteamDataSourceChange(e.target.value)}
                />
                <span>Steam Key</span>
              </label>
              <label className='radio-option'>
                <input
                  type='radio'
                  name='steamDataSource'
                  value='external-source'
                  checked={steamDataSource === 'external-source'}
                  onChange={(e) => handleSteamDataSourceChange(e.target.value)}
                />
                <span>External Source</span>
              </label>
            </div>
          </div>
        </div>

        <div className='card settings-section'>
          <h4 className='settings-section-title'>
            <DatabaseSearchIcon /> Steam Data Source
          </h4>
          <hr className='divider' />
          <div className='settings-table-steam-api-form'>
            <div data-field={steamAPIKeyHasError ? 'error' : ''}>
              <label>Steam API Key</label>
              <div className={'form-inline'}>
                <input
                  placeholder='Enter your Steam API key'
                  value={steamAPIKey}
                  onChange={(e) => setSteamAPIKey(e.target.value)}
                  aria-invalid={steamAPIKeyHasError}
                />
                <button onClick={handleSaveSteamAPIKey}>Save</button>
              </div>
              <div>
                <div className='error' role='status'>
                  Please enter a Steam API key, if you need one
                </div>
              </div>
            </div>
            {maskedSteamAPIKey && (
              <div className='settings-table-steam-api-display'>
                <span>Current API Key:</span>
                <code>{maskedSteamAPIKey}</code>
              </div>
            )}
          </div>
        </div>

        <div className='card settings-section'>
          <h4 className='settings-section-title'>
            <Eye /> Appearance
          </h4>
          <hr className='divider' />
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
    </>
  );
};

export default Settings;
