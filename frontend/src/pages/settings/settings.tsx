import './settings.scss';
import { showToast } from '@/shared/components/toast';
import type { ChangeEvent, FC } from 'react';
import { useEffect, useRef, useState } from 'react';
import { motion } from 'framer-motion';
import { Bell, BellDot, DatabaseSearchIcon, FolderOpen, Gamepad2, Globe, Plus, Rocket, Trash2, Volume2, VolumeOff } from 'lucide-react';

import {
  AddPrefix,
  GetAvailableSounds,
  GetSteamLanguages,
  LoadConfig,
  RemovePrefix,
  SetAchievementProgressUpdateMode,
  SetLanguage,
  SetNotificationSound,
  SetDesktopNotificationRenderer,
  SetSteamDataSource,
  ToggleEmulatorNotification
} from '@wa/sentinel/backend/config/file';
import { SetEnabled as SetAutostartEnabled } from '@wa/sentinel/backend/autostart/service';
import {
  GetNotificationExpireTime,
  PlaySound,
  SetupCustomNotifications,
  TestNotification,
  TestNotificationProgress
} from '@wa/sentinel/backend/notifier/service';

import {
  AchievementProgressUpdateMode,
  DesktopNotificationRenderer,
  Emulator,
  File,
  Prefix,
  SteamSource
} from '@wa/sentinel/backend/config/models';

import EmptyState from '@/shared/components/empty-state';
import { settingsContentVariants, settingsSectionVariants } from '@/pages/settings/settings-motion';

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

const emulatorDetails: Record<string, { displayName: string; searchPath: string }> = {
  'gse': { displayName: 'GSE Saves', searchPath: 'users/steamuser/AppData/Roaming/GSE Saves' },
  'goldberg-steamemu': {
    displayName: 'Goldberg SteamEmu',
    searchPath: 'users/steamuser/AppData/Roaming/Goldberg SteamEmu Saves'
  },
  'uplay-r2': {
    displayName: 'Goldberg UplayEmu',
    searchPath: 'users/steamuser/AppData/Roaming/Goldberg UplayEmu Saves'
  },
  'codex': { displayName: 'CODEX', searchPath: 'users/Public/Documents/Steam/CODEX' },
  'rune': { displayName: 'RUNE', searchPath: 'users/Public/Documents/Steam/RUNE' }
};

const Settings: FC = () => {
  const [appConfig, setAppConfig] = useState<File | null>(null);
  const [stmSrc, setStmSrc] = useState<SteamSource>();
  const [selectedLanguage, setSelectedLanguage] = useState<string>('');
  const [languages, setLanguages] = useState<{ api: string; displayName: string }[]>([]);
  const [availableSounds, setAvailableSounds] = useState<{ name: string; value: string }[]>([]);
  const [selectedSound, setSelectedSound] = useState<string>('');
  const [notificationRenderer, setNotificationRenderer] = useState<DesktopNotificationRenderer>(
    DesktopNotificationRenderer.DesktopNotificationRendererNative
  );
  const [savingRenderer, setSavingRenderer] = useState(false);
  const [notificationSetupMessage, setNotificationSetupMessage] = useState('');
  const [selectedAchievementProgressUpdateMode, setSelectedAchievementProgressUpdateMode] =
    useState<AchievementProgressUpdateMode>(AchievementProgressUpdateMode.AchievementProgressUpdateModeDefault);
  const [testNotificationDisabled, setTestNotificationDisabled] = useState(false);
  const [startOnLogin, setStartOnLogin] = useState(false);
  const [prefixToRemove, setPrefixToRemove] = useState<PrefixItem | null>(null);
  const removeDialogRef = useRef<HTMLDialogElement>(null);
  const testNotificationTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    Promise.all([loadConfig(), loadLanguages(), loadAvailableSounds()]);
  }, []);

  useEffect(() => {
    if (notificationRenderer !== DesktopNotificationRenderer.DesktopNotificationRendererCustom) {
      setNotificationSetupMessage('');
      return;
    }

    let active = true;
    SetupCustomNotifications()
      .then((message) => {
        if (active) setNotificationSetupMessage(message);
      })
      .catch((error) => {
        if (active) setNotificationSetupMessage(String(error));
      });
    return () => {
      active = false;
    };
  }, [notificationRenderer]);

  useEffect(() => {
    if (prefixToRemove && !removeDialogRef.current?.open) removeDialogRef.current?.showModal();
    if (!prefixToRemove && removeDialogRef.current?.open) removeDialogRef.current.close();
  }, [prefixToRemove]);

  useEffect(
    () => () => {
      if (testNotificationTimeout.current) clearTimeout(testNotificationTimeout.current);
    },
    []
  );

  const handleSteamDataSourceChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value as SteamSource;
    try {
      await SetSteamDataSource(value);
      setStmSrc(value);
      showToast('Steam data source updated');
    } catch (err) {
      console.error('Failed to save Steam data source:', err);
      showToast('Failed to save Steam data source', 'danger');
    }
  };

  const loadConfig = async () => {
    try {
      const cfg = await LoadConfig();
      setAppConfig(cfg);
      setStmSrc(cfg?.steamDataSource);
      setSelectedLanguage(cfg?.language?.api || 'english');
      setSelectedSound(cfg?.notificationSound || '');
      setNotificationRenderer(
        cfg?.desktop?.notificationRenderer || DesktopNotificationRenderer.DesktopNotificationRendererNative
      );
      setSelectedAchievementProgressUpdateMode(
        cfg?.achievementProgressUpdateMode || AchievementProgressUpdateMode.AchievementProgressUpdateModeDefault
      );
      setStartOnLogin(cfg?.startOnLogin ?? false);
    } catch (err) {
      showToast('Failed to load settings', 'danger');
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
      showToast(`Autostart ${newValue ? 'enabled' : 'disabled'}`);
    } catch (err) {
      showToast('Failed to update autostart setting', 'danger');
    }
  };

  const handleLanguageChange = async (e: ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    try {
      await SetLanguage(value);
      setSelectedLanguage(value);
      showToast('Language updated');
    } catch (err) {
      showToast('Failed to update language', 'danger');
    }
  };

  const handleRendererChange = async (e: ChangeEvent<HTMLSelectElement>) => {
    const renderer = e.target.value as DesktopNotificationRenderer;
    setSavingRenderer(true);
    try {
      await SetDesktopNotificationRenderer(renderer);
      setNotificationRenderer(renderer);
      showToast('Notification appearance updated');
    } catch (err) {
      showToast('Failed to save notification appearance', 'danger');
    } finally {
      setSavingRenderer(false);
    }
  };

  const handleSoundChange = async (e: ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    try {
      await SetNotificationSound(value);
      setSelectedSound(value);

      if (value) {
        await handlePlaySound(value);
      }
    } catch (err) {
      showToast('Failed to save notification sound', 'danger');
    }
  };

  const handleAchievementProgressUpdateModeChange = async (e: ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value as AchievementProgressUpdateMode;
    try {
      await SetAchievementProgressUpdateMode(value);
      setSelectedAchievementProgressUpdateMode(value);
      showToast('Achievement progress updates updated');
    } catch (err) {
      showToast('Failed to update achievement progress updates', 'danger');
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
      setTestNotificationDisabled(false);
      showToast('Failed to send test notification', 'danger');
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
      setTestNotificationDisabled(false);
      showToast('Failed to send test progress notification', 'danger');
    }
  };

  const handleToggleNotify = async (index: number) => {
    try {
      await ToggleEmulatorNotification(index);
      await loadConfig();
    } catch (err) {
      showToast('Failed to update setting', 'danger');
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
        showToast('Prefix path added');
        await Promise.all([Start(), loadConfig()]);
      }
    } catch (err) {
      console.error('Failed to add prefix:', err);
      showToast('Failed to add prefix', 'danger');
    }
  };

  const handleRemovePrefix = async (index: number) => {
    setPrefixToRemove(null);
    try {
      await RemovePrefix(index);
      await Stop();
      await Promise.all([Start(), loadConfig()]);
      showToast('Prefix removed');
    } catch (err) {
      showToast('Failed to remove prefix', 'danger');
    }
  };

  const emulators = appConfig?.emulators || [];
  const prefixes = appConfig?.prefixes || [];

  const allEmulators: EmulatorItem[] = emulators.map((emu: Emulator, index: number) => ({ emu, index }));
  const allPrefixes: PrefixItem[] = prefixes.map((prefix: Prefix, index: number) => ({ prefix, index }));

  return (
    <motion.section
      className='settings-pane page-content'
      variants={settingsContentVariants}
      initial='hidden'
      animate='visible'
    >
      <header className='settings-page-header'>
        <h2>General settings</h2>
        <p>Manage how Sentinel behaves and where it looks for games.</p>
      </header>

      <motion.div className='card settings-section settings-section-first' variants={settingsSectionVariants}>
        <header className='settings-section-header'>
          <h3 className='settings-section-title'>
            <FolderOpen /> Prefix paths
          </h3>
          <div className='settings-grid-actions'>
            <button className='outline' onClick={handleAddPrefix}>
              <Plus size={16} /> Add folder
            </button>
          </div>
        </header>
        <div className='settings-grid'>
          {allPrefixes.length === 0 ? (
            <div className='settings-empty'>
              <span className='settings-grid-icon' aria-hidden='true'>
                <FolderOpen size={20} />
              </span>
              <div className='settings-details'>
                <strong className='settings-details-label'>No prefix folders added</strong>
                <p className='settings-details-description'>Add a folder to include custom game prefix locations. Added folders will appear here.</p>
              </div>
            </div>
          ) : (
            allPrefixes.map((record) => (
              <div key={record.index} className='settings-grid-item'>
                <span className='settings-grid-icon'>
                  <FolderOpen size={20} />
                </span>
                <div className='settings-details'>
                  <strong className='settings-details-label'>Game prefix</strong>
                  <code className='settings-details-metadata' title={record.prefix.path}>{record.prefix.path}</code>
                </div>
                <div className='settings-grid-actions'>
                  <button
                    className='ghost'
                    aria-label={`Remove prefix ${record.prefix.path}`}
                    onClick={() => setPrefixToRemove(record)}
                  >
                    <Trash2 size={18} />
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      </motion.div>

      <motion.div className='card settings-section' variants={settingsSectionVariants}>
        <header className='settings-section-header'>
          <h3 className='settings-section-title'>
            <Gamepad2 /> Emulators
          </h3>
        </header>
        <div className='settings-grid'>
          {allEmulators.length === 0 ? (
            <EmptyState message='No emulators configured' />
          ) : (
            allEmulators.map((record) => (
              <div key={record.emu.id} className='settings-grid-item'>
                <span className='settings-grid-icon'>
                  <FolderOpen size={20} />
                </span>
                <div className='settings-details'>
                  <strong className='settings-details-label'>
                    {emulatorDetails[record.emu.id]?.displayName ?? record.emu.id}
                  </strong>

                  <code className='settings-details-metadata' title={emulatorDetails[record.emu.id]?.searchPath}>
                    {emulatorDetails[record.emu.id]?.searchPath ?? record.emu.id}
                  </code>
                </div>
                <div className='settings-grid-actions'>
                  <label className='switch' title='Toggle notifications for this emulator' data-tooltip-placement='left'>
                    <input
                      aria-label={`Notifications for ${record.emu.id}`}
                      type='checkbox'
                      role='switch'
                      checked={record.emu.shouldNotify}
                      onChange={() => handleToggleNotify(record.index)}
                    />
                    {record.emu.shouldNotify ? <Volume2 size={18} /> : <VolumeOff size={18} />}
                  </label>
                </div>
              </div>
            ))
          )}
        </div>
      </motion.div>

      <motion.div className='card settings-section' variants={settingsSectionVariants}>
        <header className='settings-section-header'>
          <h3 className='settings-section-title'>
            <DatabaseSearchIcon /> Steam Data Source
          </h3>
        </header>
        <div className='settings-grid'>
          <div className='settings-grid-item settings-grid-item-preference'>
            <div className='settings-details'>
              <strong className='settings-details-label'>Preferred source</strong>
              <p className='settings-details-description'>Choose where Sentinel retrieves Steam achievement information.</p>
            </div>
            <fieldset className='settings-grid-actions' aria-label='Preferred Steam data source'>
              <label className='settings-choice'>
                <input
                  type='radio'
                  name='steamDataSource'
                  value={SteamSource.Key}
                  checked={stmSrc === SteamSource.Key}
                  onChange={handleSteamDataSourceChange}
                />{' '}
                Steam
              </label>
              <label className='settings-choice'>
                <input
                  type='radio'
                  name='steamDataSource'
                  value={SteamSource.External}
                  checked={stmSrc === SteamSource.External}
                  onChange={handleSteamDataSourceChange}
                />{' '}
                External Sources
              </label>
            </fieldset>
          </div>
        </div>
      </motion.div>

      <motion.div className='card settings-section' variants={settingsSectionVariants}>
        <header className='settings-section-header'>
          <h3 className='settings-section-title'>
            <Bell /> Notifications
          </h3>
        </header>
        <div className='settings-grid'>
          <div className='settings-grid-item settings-grid-item-preference'>
            <div className='settings-details'>
              <label className='settings-details-label' htmlFor='notification-appearance'>Appearance</label>
              <p className='settings-details-description'>Choose how achievement notifications are displayed.</p>
            </div>
            <div className='settings-grid-actions settings-grid-actions-field'>
              <select
                id='notification-appearance'
                value={notificationRenderer}
                onChange={handleRendererChange}
                disabled={savingRenderer}
                aria-describedby={notificationSetupMessage ? 'notification-setup-message' : undefined}
              >
                <option value={DesktopNotificationRenderer.DesktopNotificationRendererNative}>Native (default)</option>
                <option value={DesktopNotificationRenderer.DesktopNotificationRendererCustom}>Custom</option>
              </select>
              {notificationSetupMessage && (
                <small id='notification-setup-message' className='settings-notification-setup-message'>
                  {notificationSetupMessage}
                </small>
              )}
            </div>
          </div>
          <div className='settings-grid-item settings-grid-item-preference'>
            <div className='settings-details'>
              <label className='settings-details-label' htmlFor='notification-sound'>Notification sound</label>
              <p className='settings-details-description'>Sound played when you unlock an achievement.</p>
            </div>
            <div className='settings-grid-actions'>
              <select id='notification-sound' value={selectedSound} onChange={handleSoundChange}>
                {availableSounds.map((sound) => (
                  <option key={sound.value} value={sound.value}>
                    {sound.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className='settings-grid-item settings-grid-item-preference'>
            <div className='settings-details'>
              <label className='settings-details-label' htmlFor='achievement-progress-mode'>Progress updates</label>
              <p className='settings-details-description'>Choose how achievement progress updates are delivered.</p>
            </div>
            <div className='settings-grid-actions'>
              <select
                id='achievement-progress-mode'
                value={selectedAchievementProgressUpdateMode}
                onChange={handleAchievementProgressUpdateModeChange}
              >
                {achievementProgressUpdateModes.map((mode) => (
                  <option key={mode.value} value={mode.value}>
                    {mode.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className='settings-grid-item settings-grid-item-preference'>
            <div className='settings-details'>
              <strong className='settings-details-label'>Preview notifications</strong>
              <p className='settings-details-description'>Send a test notification to check your preferences.</p>
            </div>
            <div className='settings-grid-actions'>
              <button className='outline' onClick={handleTestNotification} disabled={testNotificationDisabled}>
                <Bell size={16} /> Normal
              </button>
              <button className='outline' onClick={handleTestNotificationProgress} disabled={testNotificationDisabled}>
                <BellDot size={16} /> Progress
              </button>
            </div>
          </div>
        </div>
      </motion.div>

      <motion.div className='card settings-section' variants={settingsSectionVariants}>
        <header className='settings-section-header'>
          <h3 className='settings-section-title'>
            <Globe /> Language
          </h3>
          <span className='badge'>Coming soon</span>
        </header>
        <div className='settings-grid'>
          <div className='settings-grid-item settings-grid-item-preference'>
            <div className='settings-details'>
              <label className='settings-details-label' htmlFor='display-language'>Display language</label>
              <p className='settings-details-description'>Language selection is not available yet.</p>
            </div>
            <div className='settings-grid-actions'>
              <select id='display-language' value={selectedLanguage} onChange={handleLanguageChange} disabled>
                {languages.map((lang) => (
                  <option key={lang.api} value={lang.api}>
                    {lang.displayName}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>
      </motion.div>

      <motion.div className='card settings-section' variants={settingsSectionVariants}>
        <header className='settings-section-header'>
          <h3 className='settings-section-title'>
            <Rocket /> Startup
          </h3>
        </header>
        <div className='settings-grid'>
          <div className='settings-grid-item settings-grid-item-preference'>
            <div className='settings-details'>
              <strong className='settings-details-label'>Start on login</strong>
              <p className='settings-details-description'>Launch Sentinel minimized to the tray when you sign in.</p>
            </div>
            <div className='settings-grid-actions'>
              <label className='switch'>
                <input
                  aria-label='Start on login'
                  type='checkbox'
                  role='switch'
                  checked={startOnLogin}
                  onChange={handleStartOnLoginToggle}
                />
              </label>
            </div>
          </div>
        </div>
      </motion.div>

      <dialog
        ref={removeDialogRef}
        className='app-dialog'
        aria-labelledby='remove-prefix-title'
        onClose={() => setPrefixToRemove(null)}
      >
        <header className='app-dialog-header'>
          <h3 id='remove-prefix-title'>Remove prefix folder?</h3>
        </header>
        <div className='app-dialog-content'>
          <p>
            <strong>{prefixToRemove?.prefix.path}</strong>
          </p>
          <p>This folder will no longer be tracked by Sentinel. Files on disk will not be deleted.</p>
        </div>
        <footer className='app-dialog-footer'>
          <button className='outline' onClick={() => setPrefixToRemove(null)}>
            Cancel
          </button>
          <button
            data-variant='danger'
            onClick={() => {
              if (prefixToRemove) void handleRemovePrefix(prefixToRemove.index);
            }}
          >
            Remove folder
          </button>
        </footer>
      </dialog>
    </motion.section>
  );
};

export default Settings;
