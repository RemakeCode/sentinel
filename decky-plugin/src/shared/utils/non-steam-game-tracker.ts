import { findModuleExport } from '@decky/ui';
import { EDisplayStatus, EAppType } from '@decky/ui/dist/globals/steam-client/App';

interface NonSteamGame {
  appId: number;
  name: string;
  isRunning: boolean;
}

type AppOverviewChangeClass = {
  deserializeBinary(data: ArrayBuffer): { toObject(): any };
};

export type TrackerStatus = 'initializing' | 'ready' | 'failed';
export type TrackerCleanup = () => void;

const MAX_INIT_ATTEMPTS = 20;
const INITIAL_RETRY_DELAY_MS = 250;
const MAX_RETRY_DELAY_MS = 2000;

let cache: Map<number, NonSteamGame> = new Map();

let changeListeners: Set<() => void> = new Set();
let statusListeners: Set<(status: TrackerStatus) => void> = new Set();
let trackerStatus: TrackerStatus = 'initializing';
let retryTimer: ReturnType<typeof setTimeout> | null = null;
let cleanupAppOverview: TrackerCleanup | null = null;
let initRun = 0;

function notifyListeners() {
  changeListeners.forEach((listener) => listener());
}

function notifyStatusListeners() {
  statusListeners.forEach((listener) => listener(trackerStatus));
}

function setTrackerStatus(status: TrackerStatus) {
  if (trackerStatus === status) return;
  trackerStatus = status;
  notifyStatusListeners();
}

export function subscribeToGameChanges(callback: () => void): () => void {
  changeListeners.add(callback);
  return () => {
    changeListeners.delete(callback);
  };
}

export function subscribeToTrackerStatus(callback: (status: TrackerStatus) => void): TrackerCleanup {
  statusListeners.add(callback);
  callback(trackerStatus);
  return () => {
    statusListeners.delete(callback);
  };
}

export function getTrackerStatus(): TrackerStatus {
  return trackerStatus;
}

function resolveAppOverviewChangeClass(): AppOverviewChangeClass | null {
  try {
    return (
      (findModuleExport(
        (e: any) => typeof e?.deserializeBinary === 'function' && typeof e?.prototype?.app_overview === 'function'
      ) as AppOverviewChangeClass | undefined) ?? null
    );
  } catch (e) {
    console.error('Failed to resolve CAppOverviewChange protobuf class:', e);
    return null;
  }
}

function clearRetryTimer() {
  if (!retryTimer) return;
  clearTimeout(retryTimer);
  retryTimer = null;
}

export function processAppOverviewChange(change: any) {
  if (change.full_update) {
    cache.clear();
  }

  if (change.app_overview) {
    for (const app of change.app_overview) {
      if (app.app_type === EAppType.Shortcut) {
        const isRunning =
          app.per_client_data?.find((client: any) => client.is_available_on_current_platform)?.display_status ===
          EDisplayStatus.Running;

        cache.set(app.appid, {
          appId: app.appid,
          name: app.display_name,
          isRunning
        });
      }
    }
  }

  if (change.removed_appid) {
    for (const appId of change.removed_appid) {
      cache.delete(appId);
    }
  }

  notifyListeners();
}

export function initTracker(): TrackerCleanup {
  initRun++;
  const currentRun = initRun;
  let disposed = false;

  clearRetryTimer();
  cleanupAppOverview?.();
  cleanupAppOverview = null;
  setTrackerStatus('initializing');

  const cleanup = () => {
    if (disposed) return;
    disposed = true;
    clearRetryTimer();
    cleanupAppOverview?.();
    cleanupAppOverview = null;
  };

  const attemptInit = (attempt: number) => {
    if (disposed || currentRun !== initRun) return;

    const CAppOverviewChange = resolveAppOverviewChangeClass();
    const steamApps = globalThis.SteamClient?.Apps;
    const registerForAppOverviewChanges = steamApps?.RegisterForAppOverviewChanges;

    if (!CAppOverviewChange || typeof registerForAppOverviewChanges !== 'function') {
      if (attempt >= MAX_INIT_ATTEMPTS) {
        const missing = [
          !CAppOverviewChange ? 'CAppOverviewChange protobuf class' : null,
          typeof registerForAppOverviewChanges !== 'function' ? 'SteamClient.Apps.RegisterForAppOverviewChanges' : null
        ].filter(Boolean);

        console.error('Failed to initialize Sentinel game tracker. Missing:', missing.join(', '));
        setTrackerStatus('failed');
        return;
      }

      const delay = Math.min(INITIAL_RETRY_DELAY_MS * attempt, MAX_RETRY_DELAY_MS);
      retryTimer = setTimeout(() => attemptInit(attempt + 1), delay);
      return;
    }

    try {
      const unregister = registerForAppOverviewChanges.call(steamApps, (data: ArrayBuffer) => {
        try {
          const change = CAppOverviewChange.deserializeBinary(data).toObject();
          processAppOverviewChange(change);
        } catch (e) {
          console.error('Failed to process overview change:', e);
        }
      }) as unknown;

      cleanupAppOverview = typeof unregister === 'function' ? (unregister as TrackerCleanup) : null;
      setTrackerStatus('ready');
    } catch (e) {
      console.error('Failed to register Sentinel game tracker:', e);
      setTrackerStatus('failed');
    }
  };

  attemptInit(1);
  return cleanup;
}

export function runningGames(): NonSteamGame[] {
  return Array.from(cache.values()).filter((game) => game.isRunning);
}
