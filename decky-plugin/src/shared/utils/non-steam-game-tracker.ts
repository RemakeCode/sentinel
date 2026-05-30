import { findModuleExport } from '@decky/ui';

const EAppType_Shortcut = 1 << 30; // 1073741824 // 2^30

const EDisplayStatus_Running = 5;

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

export async function initTracker() {
  if (!CAppOverviewChange) {
    console.error('Failed to find CAppOverviewChange protobuf class');
    return;
  }

  SteamClient.Apps.RegisterForAppOverviewChanges((data: ArrayBuffer) => {
    try {
      const change = CAppOverviewChange.deserializeBinary(data).toObject();
      console.log({ change });

      if (change.full_update) {
        cache.clear();
      }

      if (change.app_overview) {
        for (const app of change.app_overview) {
          if (app.app_type === EAppType_Shortcut) {
            const isRunning = app.local_per_client_data?.display_status === EDisplayStatus_Running;

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
    } catch (e) {
      console.error('Failed to process overview change:', e);
    }
  });
}

export function runningGames(): NonSteamGame[] {
  return Array.from(cache.values()).filter((game) => game.isRunning);
}
