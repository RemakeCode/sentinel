import './settings.scss';
import type { ChangeEvent, FC } from 'react';
import { useEffect, useState } from 'react';
import { Link } from 'react-router';
import { ArrowLeft, DatabaseSearchIcon, Eye, FolderOpen, Globe, Info, Trash2, Volume2, VolumeOff } from 'lucide-react';

import {
  AddPrefix,
  GetAvailableSounds,
  GetConfig,
  GetSteamLanguages,
  RemovePrefix,
  SetLanguage,
  SetNotificationSound,
  SetSteamAPIKey,
  SetSteamDataSource,
  ToggleEmulatorNotification
} from '@wa/sentinel/backend/config/file';

import EmptyState from '@/shared/components/empty-state';
import { Emulator, File, Prefix, SteamSource } from '@wa/sentinel/backend/config/models';

import { Dialogs } from '@wailsio/runtime';
import { Header } from '@/shared/components/header/header';
import { Start, Stop } from '@wa/sentinel/backend/watcher/service';

declare global {
  interface Window {
    ot: {
      toast: (
        message: string,
        title?: string,
        options?: { variant?: 'success' | 'danger' | 'info' | 'warning' }
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

const Settings: FC = () => {
  const [appConfig, setAppConfig] = useState<File | null>(null);
  const [darkMode, setDarkMode] = useState(true);
  const [steamAPIKey, setSteamAPIKey] = useState('');
  const [steamAPIKeyHasError, setSteamAPIKeyHasError] = useState<boolean>(false);
  const [stmSrc, setStmSrc] = useState<SteamSource>();
  const [selectedLanguage, setSelectedLanguage] = useState<string>('');
  const [languages, setLanguages] = useState<{ api: string; displayName: string }[]>([]);
  const [availableSounds, setAvailableSounds] = useState<{ name: string; value: string }[]>([]);
  const [selectedSound, setSelectedSound] = useState<string>('');
  let timeout: ReturnType<typeof setTimeout>;

  useEffect(() => {
    Promise.all([loadConfig(), loadLanguages(), loadAvailableSounds()]);
  }, []);

  const handleSteamDataSourceChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value as SteamSource;
    setStmSrc(value);
    if (value === SteamSource.External) {
      try {
        await SetSteamDataSource(value);
        window.ot?.toast('Steam data source updated', 'Success', { variant: 'success' });
      } catch (err) {
        console.error('Failed to save Steam data source:', err);
        window.ot?.toast('Failed to save Steam data source', 'Error', { variant: 'danger' });
      }
    }
  };

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
      await Promise.all([SetSteamDataSource(SteamSource.Key), SetSteamAPIKey(steamAPIKey), loadConfig()]);

      window.ot?.toast('Steam API key saved', 'Success', { variant: 'success' });

      setSteamAPIKey('');
    } catch (err) {
      window.ot?.toast('Failed to save Steam API key', 'Error', { variant: 'danger' });
    }
  };

  const loadConfig = async () => {
    try {
      const cfg = await GetConfig();
      setAppConfig(cfg);
      setStmSrc(cfg?.steamDataSource);
      setSelectedLanguage(cfg?.language?.api || 'english');
      setSelectedSound(cfg?.notificationSound || '');
    } catch (err) {
      window.ot?.toast('Failed to load settings', 'Error', { variant: 'danger' });
    }
  };

  const loadLanguages = async () => {
    try {
      const langs = await GetSteamLanguages();
      setLanguages(langs);
    } catch (err) {
      console.error('Failed to load languages:', err);
    }
  };

  const loadAvailableSounds = async () => {
    try {
      const sounds = await GetAvailableSounds();
      setAvailableSounds(sounds);
    } catch (err) {
      console.error('Failed to load available sounds:', err);
    }
  };

  const handleLanguageChange = async (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    try {
      await SetLanguage(value);
      setSelectedLanguage(value);
      window.ot?.toast('Language updated', 'Success', { variant: 'success' });
      await loadConfig();
    } catch (err) {
      window.ot?.toast('Failed to update language', 'Error', { variant: 'danger' });
    }
  };

  const handleSoundChange = async (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    try {
      await SetNotificationSound(value);
      setSelectedSound(value);
      const selectedSoundOption = availableSounds.find((s) => s.value === value);
      const soundLabel = selectedSoundOption ? selectedSoundOption.name : value;
      window.ot?.toast(`Notification sound set to ${soundLabel}`, 'Success', { variant: 'success' });
      if (value) {
        await handlePlaySound(value);
      }
      await loadConfig();
    } catch (err) {
      window.ot?.toast('Failed to update notification sound', 'Error', { variant: 'danger' });
    }
  };

  const handlePlaySound = async (soundValue: string) => {
    if (!soundValue) return;
    try {
      const audio = new Audio(`/media/${soundValue}`);
      audio.onerror = () => {
        window.ot?.toast('Failed to play sound', 'Error', { variant: 'danger' });
      };
      await audio.play();
    } catch (err) {
      console.error('Failed to play sound:', err);
    }
  };

  const handleToggleNotify = async (index: number) => {
    try {
      await ToggleEmulatorNotification(index);
      await loadConfig();
    } catch (err) {
      window.ot?.toast('Failed to update setting', 'Error', { variant: 'danger' });
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
        await Stop(); // stops watcher
        window.ot?.toast('Prefix path added', 'Success', { variant: 'success' });
        await Promise.all([Start(), loadConfig()]);
      }
    } catch (err) {
      console.error('Failed to add prefix:', err);
      window.ot?.toast('Failed to add prefix', 'Error', { variant: 'danger' });
    }
  };

  const handleRemovePrefix = async (index: number) => {
    try {
      await RemovePrefix(index);
      window.ot?.toast('Prefix removed', 'Success', { variant: 'success' });
      await Promise.all([loadConfig()]);
    } catch (err) {
      window.ot?.toast('Failed to remove prefix', 'Error', { variant: 'danger' });
    }
  };

  const handleAboutDialog = async () => {
    await Dialogs.Info({
      Title: 'About Sentinel',
      Message: 'Program version 1.1.0 \n\n' + '© 2025, My Company\n\n'
    });
  };

  const handleTogglePrefix = async (index: number) => {
    console.error('TogglePrefix not implemented yet');
  };

  const emulators = appConfig?.emulators || [];
  const prefixes = appConfig?.prefixes || [];

  const allEmulators: EmulatorItem[] = emulators.map((emu: Emulator, index: number) => ({ emu, index }));
  const allPrefixes: PrefixItem[] = prefixes.map((prefix: Prefix, index: number) => ({ prefix, index }));

  return (
    <main className='full-layout'>
      <Header className='settings-header'>
        <div className='header-nav'>
          <Link to='/' viewTransition>
            <ArrowLeft />
          </Link>
          <h2>Settings</h2>
        </div>
        <div onClick={handleAboutDialog} title='About' className='settings-header-about-icon'>
          <Info size={20} />
        </div>
      </Header>
      <div className='page-content'>
        <div className='card settings-section'>
          <div className='flex justify-between items-center'>
            <h4 className='settings-section-title'>
              <FolderOpen /> <span>Prefix Paths</span>
            </h4>
            <button className='outline' onClick={handleAddPrefix}>
              <FolderOpen /> Add Prefix Folder
            </button>
          </div>
          <hr className='divider' />
          <div className='settings-grid'>
            {allPrefixes.length === 0 ? (
              <EmptyState message='No prefix paths configured' />
            ) : (
              <>
                {allPrefixes.map((record) => (
                  <div key={record.index} className='settings-grid-item'>
                    <span className='badge success'>Prefix</span>
                    <code>{record.prefix.path}</code>

                    <label className='switch'>
                      <input
                        type='checkbox'
                        role='switch'
                        checked={true}
                        disabled
                        onChange={() => handleTogglePrefix(record.index)}
                      />
                    </label>
                    <div className='settings-grid-actions' title={'Delete Prefix'}>
                      <Trash2 size={20} onClick={() => handleRemovePrefix(record.index)} />
                    </div>
                  </div>
                ))}
              </>
            )}
          </div>
        </div>

        <div className='card settings-section'>
          <div className='flex justify-between items-center'>
            <h4 className='settings-section-title'>
              <FolderOpen /> <span>Emulator Paths</span>
            </h4>
          </div>
          <hr className='divider' />
          <div className='settings-grid'>
            {allEmulators.length === 0 ? (
              <EmptyState message='No emulator paths configured' />
            ) : (
              <>
                {allEmulators.map((record) => (
                  <div key={record.index} className='settings-grid-item'>
                    <span className='badge success'>Path</span>

                    <code>{record.emu.path}</code>

                    <label className='switch' title={'Toggle Notification for this path'}>
                      <input
                        type='checkbox'
                        role='switch'
                        checked={record.emu.shouldNotify}
                        onChange={() => handleToggleNotify(record.index)}
                      />
                      {record.emu.shouldNotify ? <Volume2 size={18} /> : <VolumeOff size={18} />}
                    </label>
                    <div />
                  </div>
                ))}
              </>
            )}
          </div>
        </div>

        <div className='card settings-section'>
          <h4 className='settings-section-title'>
            <DatabaseSearchIcon /> Steam Data Source
          </h4>
          <hr className='divider' />

          <div className='settings-table-form'>
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
                  <div className='settings-table-form-display'>
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
          <div className='settings-grid'>
            <div className='settings-grid-item'>
              <span className='settings-grid-item-label'>Dark Mode</span>

              <div>
                <span className='badge'>Coming Soon</span>
              </div>
              <label className='switch'>
                <input
                  type='checkbox'
                  role='switch'
                  checked={darkMode}
                  disabled={true}
                  onChange={(e) => setDarkMode(e.target.checked)}
                />
              </label>
            </div>
          </div>
        </div>

        <div className='card settings-section'>
          <h4 className='settings-section-title'>
            <Globe /> Language
          </h4>
          <hr className='divider' />
          <div className='settings-table-form'>
            <fieldset className='hstack'>
              <legend>Preferred Language</legend>
              <label>
                <select
                  className='settings-select'
                  value={selectedLanguage}
                  onChange={handleLanguageChange}
                  disabled={true}
                >
                  {languages.map((lang: { api: string; displayName: string }) => (
                    <option key={lang.api} value={lang.api}>
                      {lang.displayName}
                    </option>
                  ))}
                </select>
              </label>
              <span className='badge'>Coming Soon</span>
            </fieldset>
          </div>
        </div>

        <div className='card settings-section'>
          <h4 className='settings-section-title'>
            <Volume2 /> Notification Sound
          </h4>
          <hr className='divider' />
          <div className='settings-table-form'>
            <fieldset className='hstack'>
              <legend>Sound Selection</legend>
              <label>
                <select className='settings-select' value={selectedSound} onChange={handleSoundChange}>
                  {availableSounds.map((sound) => (
                    <option key={sound.value} value={sound.value}>
                      {sound.name}
                    </option>
                  ))}
                </select>
              </label>
            </fieldset>
          </div>
        </div>
      </div>
    </main>
  );
};

export default Settings;
