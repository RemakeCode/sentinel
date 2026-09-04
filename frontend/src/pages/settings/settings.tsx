import './settings.scss';
import type { ChangeEvent, FC } from 'react';
import { useEffect, useRef, useState } from 'react';
import { DatabaseSearchIcon, FolderOpen, Globe, Rocket, Trash2, Volume2, VolumeOff } from 'lucide-react';

import {
  AddPrefix,
  GetAvailableSounds,
  GetSteamLanguages,
  LoadConfig,
  RemovePrefix,
  SetAchievementProgressUpdateMode,
  SetLanguage,
  SetNotificationSound,
  SetSteamDataSource,
  ToggleEmulatorNotification
} from '@wa/sentinel/backend/config/file';
import { SetEnabled as SetAutostartEnabled } from '@wa/sentinel/backend/autostart/service';
import {
  GetNotificationExpireTime,
  PlaySound,
  TestNotification,
  TestNotificationProgress
} from '@wa/sentinel/backend/notifier/service';

import { AchievementProgressUpdateMode, Emulator, File, Prefix, SteamSource } from '@wa/sentinel/backend/config/models';

import EmptyState from '@/shared/components/empty-state';

import { Dialogs } from '@wailsio/runtime';
import { Start, Stop } from '@wa/sentinel/backend/watcher/service';

interface EmulatorItem {
  emu: Emulator;
  index: number;
}

interface PrefixItem {
  prefix: Prefix;
  index: number;
}

const achievementProgressUpdateModes: { name: string; value: AchievementProgressUpdateMode }[] = [
  { name: 'Default', value: AchievementProgressUpdateMode.AchievementProgressUpdateModeDefault },
  { name: 'Silent', value: AchievementProgressUpdateMode.AchievementProgressUpdateModeSilent },
  { name: 'Disabled', value: AchievementProgressUpdateMode.AchievementProgressUpdateModeDisabled }
];

const emulatorSearchPaths: Record<string, string> = {
  'gse': 'users/steamuser/AppData/Roaming/GSE Saves',
  'goldberg-steamemu': 'users/steamuser/AppData/Roaming/Goldberg SteamEmu Saves',
  'codex': 'users/Public/Documents/Steam/CODEX',
  'rune': 'users/Public/Documents/Steam/RUNE'
};

const Settings: FC = () => {
  const [appConfig, setAppConfig] = useState<File | null>(null);
  const [stmSrc, setStmSrc] = useState<SteamSource>();
  const [selectedLanguage, setSelectedLanguage] = useState<string>('');
  const [languages, setLanguages] = useState<{ api: string; displayName: string }[]>([]);
  const [availableSounds, setAvailableSounds] = useState<{ name: string; value: string }[]>([]);
  const [selectedSound, setSelectedSound] = useState<string>('');
  const [selectedAchievementProgressUpdateMode, setSelectedAchievementProgressUpdateMode] =
    useState<AchievementProgressUpdateMode>(AchievementProgressUpdateMode.AchievementProgressUpdateModeDefault);
  const [testNotificationDisabled, setTestNotificationDisabled] = useState(false);
  const [startOnLogin, setStartOnLogin] = useState(false);
  const testNotificationTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    Promise.all([loadConfig(), loadLanguages(), loadAvailableSounds()]);
  }, []);

  const handleSteamDataSourceChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value as SteamSource;
    setStmSrc(value);
    try {
      await SetSteamDataSource(value);
      window.ot?.toast('Steam data source updated', 'Success', { variant: 'success' });
    } catch (err) {
      console.error('Failed to save Steam data source:', err);
      window.ot?.toast('Failed to save Steam data source', 'Error', { variant: 'danger' });
    }
  };

  const loadConfig = async () => {
    try {
      const cfg = await LoadConfig();
      setAppConfig(cfg);
      setStmSrc(cfg?.steamDataSource);
      setSelectedLanguage(cfg?.language?.api || 'english');
      setSelectedSound(cfg?.notificationSound || '');
      setSelectedAchievementProgressUpdateMode(
        cfg?.achievementProgressUpdateMode || AchievementProgressUpdateMode.AchievementProgressUpdateModeDefault
      );
      setStartOnLogin(cfg?.startOnLogin ?? false);
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

  const handleStartOnLoginToggle = async () => {
    const newValue = !startOnLogin;
    try {
      await SetAutostartEnabled(newValue);
      setStartOnLogin(newValue);
      window.ot?.toast(`Autostart ${newValue ? 'enabled' : 'disabled'}`, 'Success', { variant: 'success' });
    } catch (err) {
      window.ot?.toast('Failed to update autostart setting', 'Error', { variant: 'danger' });
    }
  };

  const handleLanguageChange = async (e: ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    try {
      await SetLanguage(value);
      setSelectedLanguage(value);
      window.ot?.toast('Language updated', 'Success', { variant: 'success' });
    } catch (err) {
      window.ot?.toast('Failed to update language', 'Error', { variant: 'danger' });
    }
  };

  const handleSoundChange = async (e: ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    try {
      setSelectedSound(value);
      await SetNotificationSound(value);

      if (value) {
        await handlePlaySound(value);
      }
    } catch (err) {}
  };

  const handleAchievementProgressUpdateModeChange = async (e: ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value as AchievementProgressUpdateMode;
    try {
      setSelectedAchievementProgressUpdateMode(value);
      await SetAchievementProgressUpdateMode(value);
      window.ot?.toast('Achievement progress updates updated', 'Success', { variant: 'success' });
    } catch (err) {
      window.ot?.toast('Failed to update achievement progress updates', 'Error', { variant: 'danger' });
    }
  };

  const handlePlaySound = async (soundValue: string) => {
    if (!soundValue) return;
    try {
      await PlaySound(soundValue);
    } catch (err) {
      console.warn('Failed to play sound:', err);
    }
  };

  const handleTestNotification = async () => {
    try {
      const expireTime = await GetNotificationExpireTime();
      setTestNotificationDisabled(true);
      if (testNotificationTimeout.current) {
        clearTimeout(testNotificationTimeout.current);
      }
      await TestNotification();
      testNotificationTimeout.current = setTimeout(() => setTestNotificationDisabled(false), expireTime);
    } catch (err) {
      console.error('Failed to send test notification:', err);
    }
  };

  const handleTestNotificationProgress = async () => {
    try {
      const expireTime = await GetNotificationExpireTime();
      setTestNotificationDisabled(true);
      if (testNotificationTimeout.current) {
        clearTimeout(testNotificationTimeout.current);
      }
      await TestNotificationProgress();
      testNotificationTimeout.current = setTimeout(() => setTestNotificationDisabled(false), expireTime);
    } catch (err) {
      console.error('Failed to send test progress notification:', err);
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
        ShowHiddenFiles: true,
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
      await loadConfig();
    } catch (err) {
      window.ot?.toast('Failed to remove prefix', 'Error', { variant: 'danger' });
    }
  };

  const emulators = appConfig?.emulators || [];
  const prefixes = appConfig?.prefixes || [];

  const allEmulators: EmulatorItem[] = emulators.map((emu: Emulator, index: number) => ({ emu, index }));
  const allPrefixes: PrefixItem[] = prefixes.map((prefix: Prefix, index: number) => ({ prefix, index }));

  return (
    <section className='settings-pane page-content'>
      <div className='card settings-section settings-section-first'>
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
            <FolderOpen /> <span>Emulators</span>
          </h4>
        </div>
        <hr className='divider' />
        <div className='settings-grid'>
          {allEmulators.length === 0 ? (
            <EmptyState message='No emulators configured' />
          ) : (
            <>
              {allEmulators.map((record) => (
                <div key={record.index} className='settings-grid-item'>
                  <span className='badge success'>Emulator</span>

                  <code>{emulatorSearchPaths[record.emu.id] ?? record.emu.id}</code>

                  <label className='switch' title={'Toggle Notification for this emulator'}>
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
              Steam API
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
          {/* TODO: restore API key input and masked key display if Steam ever requires key auth */}
        </div>
      </div>

      <div className='card settings-section'>
        <h4 className='settings-section-title'>
          <Volume2 /> Notification
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
          <fieldset className='hstack'>
            <legend>Achievement Progress Updates</legend>
            <label>
              <select
                className='settings-select'
                value={selectedAchievementProgressUpdateMode}
                onChange={handleAchievementProgressUpdateModeChange}
              >
                {achievementProgressUpdateModes.map((mode) => (
                  <option key={mode.value} value={mode.value}>
                    {mode.name}
                  </option>
                ))}
              </select>
            </label>
          </fieldset>
          <fieldset className='hstack'>
            <legend>Test Notification</legend>
            <div className='hstack'>
              <button className='outline' onClick={handleTestNotification} disabled={testNotificationDisabled}>
                Normal
              </button>
              <button className='outline' onClick={handleTestNotificationProgress} disabled={testNotificationDisabled}>
                Progress
              </button>
            </div>
          </fieldset>
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
          <Rocket /> Startup
        </h4>
        <hr className='divider' />
        <div className='settings-grid'>
          <div className='settings-grid-item'>
            <span className='badge success'>Autostart</span>
            <span>Start on login (minimized to tray)</span>
            <label className='switch' title='Toggle autostart on login'>
              <input type='checkbox' role='switch' checked={startOnLogin} onChange={handleStartOnLoginToggle} />
            </label>
            <div />
          </div>
        </div>
      </div>
    </section>
  );
};

export default Settings;
