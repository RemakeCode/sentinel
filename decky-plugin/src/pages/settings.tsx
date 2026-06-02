import { type FC, useEffect, useRef, useState } from 'react';
import {
  DialogBody,
  DialogButton,
  DialogControlsSection,
  DialogControlsSectionHeader,
  DialogLabel,
  Dropdown,
  Field,
  SidebarNavigation,
  TextField,
  Toggle
} from '@decky/ui';
import { openFilePicker, toaster } from '@decky/api';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import { usePlayAudio } from '@/shared/utils/usePlayAudio';
import { clearMapping, type GameMapping, getAllMappings } from '@/shared/utils/game-mappings';
import { showConfirmModal } from '@/shared/utils/confirm';
import { BsTrash } from 'react-icons/bs';
import { FaVolumeHigh, FaVolumeOff } from 'react-icons/fa6';
import { FaBook, FaCircle, FaCog, FaLink, FaSave } from 'react-icons/fa';

const fetcher = new Fetcher();

interface Prefix {
  path: string;
}

interface Emulator {
  path: string;
  shouldNotify: boolean;
}

interface AppConfig {
  prefixes: Prefix[];
  emulators: Emulator[];
  steamDataSource: string;
  steamApiKeyMasked: string;
  notificationSound: string;
  logLevel: string;
}

interface SoundOption {
  name: string;
  value: string;
}

const MappingsContent: FC = () => {
  const [mappings, setMappings] = useState<Record<number, GameMapping>>({});

  const loadMappings = () => {
    setMappings(getAllMappings());
  };

  useEffect(() => {
    loadMappings();
  }, []);

  const handleDelete = async (nonSteamAppId: number, sentinelName: string) => {
    const confirmed = await showConfirmModal({
      title: 'Delete Mapping',
      description: `Are you sure you want to delete the mapping for ${sentinelName}?`,
      okText: "Yes, I'm sure",
      cancelText: 'Cancel',
      destructive: true
    });
    if (!confirmed) return;
    clearMapping(nonSteamAppId);
    loadMappings();
  };

  const entries = Object.entries(mappings).sort(([, a], [, b]) => b.createdAt - a.createdAt);

  return (
    <DialogControlsSection>
      <DialogControlsSectionHeader>Game Mappings</DialogControlsSectionHeader>
      {entries.length === 0 ? (
        <Field label='No mappings saved yet.' />
      ) : (
        entries.map(([appIdStr, mapping]) => {
          const nonSteamAppId = Number(appIdStr);
          return (
            <Field key={nonSteamAppId} label={mapping.shortcutName} description={mapping.sentinelName}>
              <DialogButton onClick={() => handleDelete(nonSteamAppId, mapping.sentinelName)} style={{ minWidth: 0 }}>
                <BsTrash />
              </DialogButton>
            </Field>
          );
        })
      )}
    </DialogControlsSection>
  );
};

const SettingsPage: FC = () => {
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [availableSounds, setAvailableSounds] = useState<SoundOption[]>([]);
  const [steamAPIKey, setSteamAPIKey] = useState('');
  const [steamAPIKeyHasError, setSteamAPIKeyHasError] = useState(false);
  const [stmSrc, setStmSrc] = useState<string>('');
  const [testNotificationDisabled, setTestNotificationDisabled] = useState(false);
  const [serviceStatus, setServiceStatus] = useState<'loading' | 'online' | 'offline'>('loading');
  const steamAPIKeyTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const testNotificationTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { play } = usePlayAudio();

  const loadConfig = async () => {
    try {
      const cfg = await fetcher.get<AppConfig>(`${BASE_URL}/config`);
      setConfig(cfg);
      setStmSrc(cfg.steamDataSource);
    } catch {
      // config load failed
    }
  };

  const loadAvailableSounds = async () => {
    try {
      const sounds = await fetcher.get<SoundOption[]>(`${BASE_URL}/config/available-sounds`);
      setAvailableSounds(sounds);
    } catch {
      // sounds failed to load
    }
  };

  const checkServiceStatus = async () => {
    try {
      await fetcher.get(`${BASE_URL}/ready`);
      setServiceStatus('online');
    } catch {
      setServiceStatus('offline');
    }
  };

  useEffect(() => {
    Promise.all([loadConfig(), loadAvailableSounds(), checkServiceStatus()]);
  }, []);

  const handleSteamDataSourceChange = async (source: string) => {
    setStmSrc(source);
    if (source === 'external') {
      try {
        await fetcher.put(`${BASE_URL}/config/steam-data-source`, { source: 'external' });
        await loadConfig();
      } catch {
        // failed to save data source
      }
    }
  };

  const handleSaveSteamAPIKey = async () => {
    if (steamAPIKeyTimeout.current) {
      clearTimeout(steamAPIKeyTimeout.current);
    }

    if (!steamAPIKey) {
      setSteamAPIKeyHasError(true);
      steamAPIKeyTimeout.current = setTimeout(() => setSteamAPIKeyHasError(false), 5000);
      return;
    }
    try {
      await fetcher.put(`${BASE_URL}/config/steam-api-key`, { apiKey: steamAPIKey });
      await fetcher.put(`${BASE_URL}/config/steam-data-source`, { source: 'key' });
      await loadConfig();
      setSteamAPIKey('');
      toaster.toast({ title: 'Success', body: 'Steam API key saved' });
    } catch {
      toaster.toast({ title: 'Error', body: 'Failed to save Steam API key' });
    }
  };

  const handleAddPrefix = async () => {
    const result = await openFilePicker(1, '/', false, true);
    if (result.realpath) {
      try {
        await fetcher.post(`${BASE_URL}/config/prefix`, { path: result.realpath });
        await loadConfig();
        toaster.toast({ title: 'Success', body: 'Prefix path added' });
      } catch {
        toaster.toast({ title: 'Error', body: 'Failed to add prefix' });
      }
    }
  };

  const handleRemovePrefix = async (index: number) => {
    try {
      await fetcher.delete(`${BASE_URL}/config/prefix/${index}`);
      await loadConfig();
      toaster.toast({ title: 'Success', body: 'Prefix removed' });
    } catch {
      toaster.toast({ title: 'Error', body: 'Failed to remove prefix' });
    }
  };

  const handleToggleNotify = async (index: number) => {
    try {
      await fetcher.patch(`${BASE_URL}/config/emulator-notification/${index}`);
      await loadConfig();
    } catch {
      toaster.toast({ title: 'Error', body: 'Failed to update notification setting' });
    }
  };

  const handleSoundChange = async (option: { data: string }) => {
    const value = option.data;
    try {
      await fetcher.post(`${BASE_URL}/config/notification-sound`, { sound: value });
      if (value) {
        await play(value);
      }
    } catch {
      // failed to set sound
    }
  };

  const handleTestNotification = async () => {
    setTestNotificationDisabled(true);
    if (testNotificationTimeout.current) clearTimeout(testNotificationTimeout.current);
    try {
      await fetcher.post(`${BASE_URL}/notifications/test`, {});
      testNotificationTimeout.current = setTimeout(() => setTestNotificationDisabled(false), 7000);
    } catch {
      setTestNotificationDisabled(false);
      toaster.toast({ title: 'Error', body: 'Failed to send test notification' });
    }
  };

  const handleTestNotificationProgress = async () => {
    setTestNotificationDisabled(true);
    if (testNotificationTimeout.current) clearTimeout(testNotificationTimeout.current);
    try {
      await fetcher.post(`${BASE_URL}/notifications/test-progress`, {});
      testNotificationTimeout.current = setTimeout(() => setTestNotificationDisabled(false), 7000);
    } catch {
      setTestNotificationDisabled(false);
      toaster.toast({ title: 'Error', body: 'Failed to send test progress notification' });
    }
  };

  const handleLoggingToggle = async (enabled: boolean) => {
    try {
      await fetcher.put(`${BASE_URL}/config/logging`, { enabled });
      setConfig((prev) => (prev ? { ...prev, logLevel: enabled ? 'info' : 'off' } : prev));
    } catch {
      toaster.toast({ title: 'Error', body: 'Failed to update logging setting' });
    }
  };

  const prefixes = config?.prefixes || [];
  const emulators = config?.emulators || [];

  return (
    <SidebarNavigation
      title='Settings'
      pages={[
        {
          title: 'Settings',
          identifier: 'settings',
          icon: <FaCog />,
          content: (
            <DialogBody>
              <DialogControlsSection>
                <DialogControlsSectionHeader>Prefix Paths</DialogControlsSectionHeader>
                {prefixes.length === 0 ? (
                  <Field label='No prefix paths configured' />
                ) : (
                  prefixes.map((prefix, index) => (
                    <Field key={index} label={prefix.path}>
                      <DialogButton onClick={() => handleRemovePrefix(index)} focusable style={{ minWidth: 0 }}>
                        <BsTrash />
                      </DialogButton>
                    </Field>
                  ))
                )}
                <Field label=''>
                  <DialogButton onClick={handleAddPrefix}>Add Prefix Folder</DialogButton>
                </Field>
              </DialogControlsSection>
              <DialogControlsSection>
                <DialogControlsSectionHeader>Emulator Paths</DialogControlsSectionHeader>
                {emulators.length === 0 ? (
                  <Field label='No emulator paths configured' />
                ) : (
                  emulators.map((emu, index) => (
                    <Field
                      key={index}
                      label={emu.path}
                      icon={
                        <div style={{ display: 'block' }}>{emu.shouldNotify ? <FaVolumeHigh /> : <FaVolumeOff />}</div>
                      }
                    >
                      <Toggle value={emu.shouldNotify} onChange={() => handleToggleNotify(index)} />
                    </Field>
                  ))
                )}
              </DialogControlsSection>
              <DialogControlsSection>
                <DialogControlsSectionHeader>Steam Data Source</DialogControlsSectionHeader>
                <Field label='Data source' childrenContainerWidth='fixed'>
                  <Dropdown
                    rgOptions={[
                      { data: 'external', label: 'External Source' },
                      { data: 'key', label: 'Steam Key' }
                    ]}
                    selectedOption={stmSrc}
                    onChange={(option) => handleSteamDataSourceChange(option.data)}
                  />
                </Field>
                {stmSrc === 'key' && (
                  <Field
                    label='API Key'
                    description={
                      <>
                        {steamAPIKeyHasError && <span>Please enter a Steam API key</span>}
                        {config?.steamApiKeyMasked && <span>Current: {config.steamApiKeyMasked}</span>}
                      </>
                    }
                  >
                    <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                      <TextField
                        style={{ minWidth: '20rem', width: '20rem' }}
                        value={steamAPIKey}
                        onChange={(e) => {
                          setSteamAPIKey(e.target.value);
                          setSteamAPIKeyHasError(false);
                        }}
                      />

                      <DialogButton style={{ minWidth: 0 }} onClick={handleSaveSteamAPIKey}>
                        <FaSave />
                      </DialogButton>
                    </div>
                  </Field>
                )}
              </DialogControlsSection>
              <DialogControlsSection>
                <DialogControlsSectionHeader>Notification Sound</DialogControlsSectionHeader>
                <Field label='Sound' childrenContainerWidth='fixed'>
                  <Dropdown
                    rgOptions={availableSounds.map((s) => ({ data: s.value, label: s.name }))}
                    selectedOption={config?.notificationSound || ''}
                    onChange={handleSoundChange}
                    strDefaultLabel='Select a sound'
                  />
                </Field>
              </DialogControlsSection>
              <DialogControlsSection>
                <DialogControlsSectionHeader>Test Notification</DialogControlsSectionHeader>
                <Field label='Normal'>
                  <DialogButton disabled={testNotificationDisabled} onClick={handleTestNotification}>
                    Test
                  </DialogButton>
                </Field>
                <Field label='Progress'>
                  <DialogButton disabled={testNotificationDisabled} onClick={handleTestNotificationProgress}>
                    Test
                  </DialogButton>
                </Field>
              </DialogControlsSection>
              <DialogControlsSection>
                <DialogControlsSectionHeader>Service Status</DialogControlsSectionHeader>
                <Field
                  label='Backend'
                  icon={
                    <FaCircle
                      fill={
                        serviceStatus === 'online' ? '#22c55e' : serviceStatus === 'offline' ? '#ef4444' : '#6b7280'
                      }
                    />
                  }
                >
                  <DialogLabel>{serviceStatus}</DialogLabel>
                </Field>
              </DialogControlsSection>
            </DialogBody>
          )
        },

        {
          title: 'Game Mappings',
          identifier: 'mappings',
          icon: <FaLink />,
          content: <MappingsContent />
        },

        {
          title: 'Logging',
          identifier: 'logging',
          icon: <FaBook />,
          content: (
            <DialogBody>
              <DialogControlsSection>
                <DialogControlsSectionHeader>Logging</DialogControlsSectionHeader>
                <Field label='Enable logging'>
                  <Toggle value={(config?.logLevel || '') === 'info'} onChange={handleLoggingToggle} />
                </Field>
              </DialogControlsSection>
            </DialogBody>
          )
        }
      ]}
    />
  );
};

export default SettingsPage;
