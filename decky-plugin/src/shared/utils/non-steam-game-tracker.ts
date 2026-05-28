import { findModuleExport } from '@decky/ui';

const EAppType_Shortcut = 1 << 30; // 1073741824

const EDisplayStatus_Running = 5;

interface NonSteamGame {
  appId: number;
  name: string;
  isRunning: boolean;
}

let cachedNonSteamGames: Map<number, NonSteamGame> = new Map();
let changeListeners: Set<() => void> = new Set();

function notifyListeners() {
  changeListeners.forEach((listener) => listener());
}

export function subscribeToNonSteamGameChanges(callback: () => void): () => void {
  changeListeners.add(callback);
  return () => {
    changeListeners.delete(callback);
  };
}

const CAppOverviewChange = findModuleExport(
  (e: any) => typeof e?.deserializeBinary === 'function' && typeof e?.prototype?.app_overview === 'function'
) as { deserializeBinary(data: ArrayBuffer): { toObject(): any } };

export async function initNonSteamGameTracker() {
  if (!CAppOverviewChange) {
    console.error('Failed to find CAppOverviewChange protobuf class');
    return;
  }

  SteamClient.Apps.RegisterForAppOverviewChanges((data: ArrayBuffer) => {
    try {
      const change = CAppOverviewChange.deserializeBinary(data).toObject();
      console.log({ change });

      if (change.full_update) {
        cachedNonSteamGames.clear();
      }

      if (change.app_overview) {
        for (const app of change.app_overview) {
          if (app.app_type === EAppType_Shortcut) {
            const isRunning = app.local_per_client_data?.display_status === EDisplayStatus_Running;

            cachedNonSteamGames.set(app.appid, {
              appId: app.appid,
              name: app.display_name,
              isRunning
            });
          }
        }
      }

      if (change.removed_appid) {
        for (const appId of change.removed_appid) {
          cachedNonSteamGames.delete(appId);
        }
      }

      notifyListeners();
    } catch (e) {
      console.error('Failed to process overview change:', e);
    }
  });
}

export function getRunningNonSteamGames(): NonSteamGame[] {
  return Array.from(cachedNonSteamGames.values()).filter((game) => game.isRunning);
}
