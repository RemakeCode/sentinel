import { findModuleExport } from '@decky/ui';
import { EDisplayStatus, EAppType } from '@decky/ui/dist/globals/steam-client/App';
import type {SteamClient} from '@decky/ui/dist/globals/steam-client';

let SteamClient: SteamClient;

interface NonSteamGame {
  appId: number;
  name: string;
  isRunning: boolean;
}

let cache: Map<number, NonSteamGame> = new Map();

let changeListeners: Set<() => void> = new Set();

function notifyListeners() {
  changeListeners.forEach((listener) => listener());
}

export function subscribeToGameChanges(callback: () => void): () => void {
  changeListeners.add(callback);
  return () => {
    changeListeners.delete(callback);
  };
}

const CAppOverviewChange = findModuleExport(
  (e: any) => typeof e?.deserializeBinary === 'function' && typeof e?.prototype?.app_overview === 'function'
) as { deserializeBinary(data: ArrayBuffer): { toObject(): any } };

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

export async function initTracker() {
  if (!CAppOverviewChange) {
    console.error('Failed to find CAppOverviewChange protobuf class');
    return;
  }

  SteamClient.Apps.RegisterForAppOverviewChanges((data: ArrayBuffer) => {
    try {
      const change = CAppOverviewChange.deserializeBinary(data).toObject();
      processAppOverviewChange(change);
    } catch (e) {
      console.error('Failed to process overview change:', e);
    }
  });
}

export function runningGames(): NonSteamGame[] {
  return Array.from(cache.values()).filter((game) => game.isRunning);
}
