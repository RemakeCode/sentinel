import React, { useEffect, useState } from 'react';
import { Link } from 'react-router';
import { ArrowLeft, DatabaseSearchIcon, Eye, FolderOpen, Trash2, Volume2, VolumeOff } from 'lucide-react';

import {
  AddEmulator,
  AddPrefix,
  LoadConfig,
  RemoveEmulator,
  RemovePrefix,
  SetSteamAPIKey,
  SetSteamDataSource,
  ToggleEmulatorNotification
} from '@wa/sentinel/backend/config/file';

import './settings.scss';
import EmptyState from '@/shared/components/EmptyState';
import { Emulator, File, Prefix, SteamSource } from '@wa/sentinel/backend/config/models';

import { Dialogs } from '@wailsio/runtime';
import { Header } from '@/shared/components/Header/Header';

declare global {
  interface Window {
    ot: {
      toast: (
        message: string,
        title?: string,
        options?: { variant?: 'success' | 'error' | 'info' | 'warning' }
      ) => void;
    };
  }
}

interface EmulatorItem {
  emu: Emulator;
  index: number;
}

interface PrefixItem {
  prefix: Prefix;
  index: number;
}

const Settings: React.FC = () => {
  const [appConfig, setAppConfig] = useState<File | null>(null);
  const [darkMode, setDarkMode] = useState(true);
  const [steamAPIKey, setSteamAPIKey] = useState('');
  const [steamAPIKeyHasError, setSteamAPIKeyHasError] = useState<boolean>(false);
  const [stmSrc, setStmSrc] = useState<SteamSource>();

  useEffect(() => {
    loadConfig();
  }, []);

  const handleSteamDataSourceChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value as SteamSource;
    setStmSrc(value);
    if (value === SteamSource.External) {
      try {
        await SetSteamDataSource(value);
        window.ot?.toast('Steam data source updated', 'Success', { variant: 'success' });
      } catch (err) {
        console.error('Failed to save Steam data source:', err);
        window.ot?.toast('Failed to save Steam data source', 'Error', { variant: 'error' });
      }
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
      await Promise.all([SetSteamDataSource(SteamSource.Key), SetSteamAPIKey(steamAPIKey)]);

      window.ot?.toast('Steam API key saved', 'Success', { variant: 'success' });

      await loadConfig();
      //Clear input field
      setSteamAPIKey('');
    } catch (err) {
      console.error('Failed to save Steam API key:', err);
      window.ot?.toast('Failed to save Steam API key', 'Error', { variant: 'error' });
    }
  };

  const loadConfig = async () => {
    try {
      const cfg = await LoadConfig();
      setAppConfig(cfg);
      setStmSrc(cfg?.steamDataSource);
    } catch (err) {
      window.ot?.toast('Failed to load settings', 'Error', { variant: 'error' });
    }
  };

  const handleAddEmulator = async () => {
    try {
      const selectedPath = await Dialogs.OpenFile({
        CanChooseDirectories: true,
        CanChooseFiles: false,
        Title: 'Select A Folder to Watch'
      });

      if (selectedPath) {
        await AddEmulator(selectedPath);
        window.ot?.toast('Emulator path added', 'Success', { variant: 'success' });
        await loadConfig();
      }
    } catch (err) {
      console.error('Failed to add emulator:', err);
      window.ot?.toast('Failed to add emulator', 'Error', { variant: 'error' });
    }
  };

  const handleToggleNotify = async (index: number) => {
    try {
      await ToggleEmulatorNotification(index);
      await loadConfig();
    } catch (err) {
      window.ot?.toast('Failed to update setting', 'Error', { variant: 'error' });
    }
  };

  const handleRemoveEmulator = async (index: number) => {
    try {
      await RemoveEmulator(index);
      window.ot?.toast('Emulator removed', 'Success', { variant: 'success' });
      await loadConfig();
    } catch (err) {
      console.error('Failed to remove emulator:', err);
      window.ot?.toast('Failed to remove emulator', 'Error', { variant: 'error' });
    }
  };

  const handleAddPrefix = async () => {
    try {
      const selectedPath = await Dialogs.OpenFile({
        CanChooseDirectories: true,
        CanChooseFiles: false,
        Title: 'Select A Prefix Folder'
      });

      if (selectedPath) {
        await AddPrefix(selectedPath);
        window.ot?.toast('Prefix path added', 'Success', { variant: 'success' });
        await loadConfig();
      }
    } catch (err) {
      console.error('Failed to add prefix:', err);
      window.ot?.toast('Failed to add prefix', 'Error', { variant: 'error' });
    }
  };

  const handleRemovePrefix = async (index: number) => {
    try {
      await RemovePrefix(index);
      window.ot?.toast('Prefix removed', 'Success', { variant: 'success' });
      await loadConfig();
    } catch (err) {
      console.error('Failed to remove prefix:', err);
      window.ot?.toast('Failed to remove prefix', 'Error', { variant: 'error' });
    }
  };

  const emulators = appConfig?.emulators || [];
  const prefixes = appConfig?.prefixes || [];

  const allEmulators: EmulatorItem[] = emulators.map((emu: Emulator, index: number) => ({ emu, index }));
  const allPrefixes: PrefixItem[] = prefixes.map((prefix: Prefix, index: number) => ({ prefix, index }));

  return (
    <main className='full-layout'>
      <Header className='settings-header'>
        <Link to='/' viewTransition>
          <ArrowLeft />
        </Link>
        <h2>Settings</h2>
      </Header>
      <div className='main-content'>
        <div className='card settings-section'>
          <div className='flex justify-between items-center'>
            <h4 className='settings-section-title'>
              <FolderOpen /> <span>Prefix Paths</span>
            </h4>
            <button data-variant='primary' onClick={handleAddPrefix}>
              <FolderOpen /> Add Prefix Folder
            </button>
          </div>
          <hr className='divider' />
          <div className='settings-table'>
            {allPrefixes.length === 0 ? (
              <EmptyState message='No prefix paths configured' />
            ) : (
              <table>
                <tbody>
                  {allPrefixes.map((record) => (
                    <tr key={record.index}>
                      <td className='settings-table-cell-type'>
                        <span className='badge'>Prefix</span>
                      </td>
                      <td className='settings-table-cell-path'>
                        <code>{record.prefix.path}</code>
                      </td>
                      <td className='settings-table-cell-actions'>
                        <div className='settings-table-actions hstack gap-4'>
                          <Trash2 onClick={() => handleRemovePrefix(record.index)} />
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

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
                        {record.emu.isDefault ? (
                          <span className='badge'>Default</span>
                        ) : (
                          <span className='badge success'>Custom</span>
                        )}
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
                            {record.emu.shouldNotify ? (
                              <Volume2 width={20} fill='var(--success)' />
                            ) : (
                              <VolumeOff width={20} fill='var(--danger)' />
                            )}
                          </label>
                          |
                          {
                            <Trash2
                              onClick={() => handleRemoveEmulator(record.index)}
                              className={record.emu.isDefault ? 'disabled' : ''}
                            />
                          }
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

        <div className='card settings-section'>
          <h4 className='settings-section-title'>
            <DatabaseSearchIcon /> Steam Data Source
          </h4>
          <hr className='divider' />

          <div className='settings-table-steam-api-form'>
            <fieldset className='hstack'>
              <legend>Preference</legend>
              <label className='radio-option'>
                <input
                  type='radio'
                  name='steamDataSource'
                  value={SteamSource.Key}
                  checked={stmSrc === SteamSource.Key}
                  onChange={handleSteamDataSourceChange}
                />
                Steam Key
              </label>
              <label className='radio-option'>
                <input
                  type='radio'
                  name='steamDataSource'
                  value={SteamSource.External}
                  checked={stmSrc === SteamSource.External}
                  onChange={handleSteamDataSourceChange}
                />
                External Source
              </label>
            </fieldset>
            {stmSrc === SteamSource.Key && (
              <>
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
                {appConfig?.steamApiKeyMasked && (
                  <div className='settings-table-steam-api-display'>
                    <span>Current API Key:</span>
                    <code>{appConfig.steamApiKeyMasked}</code>
                  </div>
                )}
              </>
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
    </main>
  );
};

export default Settings;
