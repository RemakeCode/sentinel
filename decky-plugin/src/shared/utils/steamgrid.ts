import type { GameBasics } from '@/shared/types/GameBasics';
import { getAllMappings, setMapping, type GameMapping } from '@/shared/utils/game-mappings';
import { matchGameByName } from '@/shared/utils/game-matcher';
import { nonSteamGames, type NonSteamGame } from '@/shared/utils/non-steam-game-tracker';

interface AppConfig {
  decky?: {
    UseSteamGrid: boolean;
  };
}

interface DeckyGameBasics extends GameBasics {
  FallbackPortraitImage?: string;
}

function populateMissingMappings(shortcuts: NonSteamGame[], games: GameBasics[]): Record<number, GameMapping> {
  const mappings = getAllMappings();

  for (const shortcut of shortcuts) {
    if (mappings[shortcut.appId]) {
      continue;
    }

    const match = matchGameByName(shortcut.name, games);
    if (!match) {
      continue;
    }

    setMapping(shortcut.appId, match.AppID, match.Name, shortcut.name);
    mappings[shortcut.appId] = {
      sentinelAppId: match.AppID,
      sentinelName: match.Name,
      shortcutName: shortcut.name,
      createdAt: Date.now()
    };
  }

  return mappings;
}

function findShortcutAppIdForGame(
  game: GameBasics,
  mappings: Record<number, GameMapping>,
  shortcutIds: Set<number>
): number | null {
  for (const [shortcutAppId, mapping] of Object.entries(mappings)) {
    const parsedShortcutAppId = Number(shortcutAppId);
    if (mapping.sentinelAppId === game.AppID && shortcutIds.has(parsedShortcutAppId)) {
      return parsedShortcutAppId;
    }
  }

  return null;
}

function decorateGames(config: AppConfig, games: GameBasics[]): DeckyGameBasics[] {
  if (!config.decky?.UseSteamGrid) {
    return games;
  }

  const shortcuts = nonSteamGames();
  const shortcutIds = new Set(shortcuts.map((game) => game.appId));
  const mappings = populateMissingMappings(shortcuts, games);

  return games.map((game) => {
    const shortcutAppId = findShortcutAppIdForGame(game, mappings, shortcutIds);
    if (!shortcutAppId) {
      return game;
    }

    return {
      ...game,
      FallbackPortraitImage: game.PortraitImage,
      PortraitImage: `/api/media/steamgrid/${shortcutAppId}/portrait`
    };
  });
}

export type { AppConfig, DeckyGameBasics };
export { decorateGames };
