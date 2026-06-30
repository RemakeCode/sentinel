import { type FC, useCallback, useEffect, useRef, useState } from 'react';
import {
  DialogBody,
  DialogHeader,
  Focusable,
  Menu,
  MenuItem,
  Navigation,
  Spinner,
  showContextMenu
} from '@decky/ui';
import { toaster } from '@decky/api';
import { LibraryImage } from '@/shared/components/library-image';
import { EmptyState } from '@/shared/components/empty-state';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import { computeProgress } from '@/shared/utils/utils';
import type { GameBasics } from '@/shared/types/GameBasics';
import { styles } from '@/shared/styles';
import { getAllMappings, setMapping, type GameMapping } from '@/shared/utils/game-mappings';
import { matchGameByName } from '@/shared/utils/game-matcher';
import { nonSteamGames, type NonSteamGame } from '@/shared/utils/non-steam-game-tracker';

//language=css
const libraryStyles = `
  .sentinel-library-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 24px;
  }

  .sentinel-library-header {
    font-size: 24px;
  }

  .sentinel-library-loader {
    width: 100%;
    min-height: 60vh;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .sentinel-library-loader svg {
    width: 48px;
    height: 48px;
  }

  .sentinel-library-sync {
    width: max-content;
    height: 25px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #1c1f24;
    position: fixed;
    transform: translate(-50%, -50%);
    left: 50%;
    padding-inline: 8px;
  }

  .sentinel-library-sync-meta {
    display: flex;
    align-items: center;
    gap: 12px;
    justify-content: space-between;
    font-size: 13px;
    font-weight: 700;
    letter-spacing: 0.03em;
    color: var(--gpColor-Blue, #1a9fff);
  }
`;

const fetcher = new Fetcher();
const SYNC_POLL_INTERVAL_MS = 1000;

interface LibrarySyncStatus {
  State: string;
  Current: number;
  Total: number;
}

interface AppConfig {
  decky?: {
    UseSteamGrid: boolean;
  };
}

interface DeckyGameBasics extends GameBasics {
  FallbackPortraitImage?: string;
}

const emptySyncStatus: LibrarySyncStatus = { State: 'idle', Current: 0, Total: 0 };

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

const LibraryPage: FC = () => {
  const [games, setGames] = useState<DeckyGameBasics[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncStatus, setSyncStatus] = useState<LibrarySyncStatus>(emptySyncStatus);
  const [refreshingGameIds, setRefreshingGameIds] = useState<string[]>([]);
  const lastSyncStatusRef = useRef<LibrarySyncStatus>(emptySyncStatus);

  const loadGames = useCallback(async (showLoading = false, clearOnError = false) => {
    if (showLoading) {
      setLoading(true);
    }

    try {
      const [config, data] = await Promise.all([
        fetcher.get<AppConfig>(`${BASE_URL}/config`),
        fetcher.get<GameBasics[]>(`${BASE_URL}/games`)
      ]);
      const decoratedGames = decorateGames(config, data);
      setGames(decoratedGames);
      return decoratedGames;
    } catch {
      if (clearOnError) {
        setGames([]);
      }
      return [];
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    let active = true;
    let intervalId: ReturnType<typeof setInterval> | undefined;

    const loadSyncStatus = async () => {
      try {
        return await fetcher.get<LibrarySyncStatus>(`${BASE_URL}/games/sync-status`);
      } catch {
        return null;
      }
    };

    const pollSyncStatus = async () => {
      const syncStatus = await loadSyncStatus();
      if (!active || !syncStatus) {
        return;
      }

      const previous = lastSyncStatusRef.current;
      const syncStarted = previous.State !== 'running' && syncStatus.State === 'running';
      const progressed =
        syncStatus.State === 'running' && (syncStarted ? syncStatus.Current > 0 : syncStatus.Current > previous.Current);
      const reachedTerminalState =
        (syncStatus.State === 'done' || syncStatus.State === 'error') &&
        (previous.State !== syncStatus.State ||
          previous.Current !== syncStatus.Current ||
          previous.Total !== syncStatus.Total);

      lastSyncStatusRef.current = syncStatus;
      setSyncStatus(syncStatus);

      if (progressed || reachedTerminalState) {
        await loadGames(false);
      }
    };

    void loadGames(true, true);
    void pollSyncStatus();
    intervalId = setInterval(pollSyncStatus, SYNC_POLL_INTERVAL_MS);

    return () => {
      active = false;
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }, [loadGames]);

  const handleRefreshGame = async (appId: string) => {
    if (!appId || refreshingGameIds.includes(appId)) {
      return;
    }

    setRefreshingGameIds((current) => [...current, appId]);

    try {
      const [config, refreshedGame] = await Promise.all([
        fetcher.get<AppConfig>(`${BASE_URL}/config`),
        fetcher.post<GameBasics>(`${BASE_URL}/games/${appId}/refresh`, {})
      ]);
      const decoratedGame = decorateGames(config, [refreshedGame])[0];
      setGames((current) =>
        current.map((game) => {
          if (game.AppID !== decoratedGame.AppID) {
            return game;
          }

          return decoratedGame;
        })
      );
      toaster.toast({ title: 'Success', body: `${refreshedGame.Name || 'Game'} refreshed` });
    } catch {
      toaster.toast({ title: 'Error', body: 'Failed to refresh game' });
    } finally {
      setRefreshingGameIds((current) => current.filter((id) => id !== appId));
    }
  };

  const openGameContextMenu = (appId: string, parent?: EventTarget | null) => {
    showContextMenu(
      <Menu label='Game Actions'>
        <MenuItem
          disabled={refreshingGameIds.includes(appId)}
          onClick={() => {
            void handleRefreshGame(appId);
          }}
        >
          {refreshingGameIds.includes(appId) ? 'Refreshing...' : 'Refresh game'}
        </MenuItem>
      </Menu>,
      parent ?? undefined
    );
  };

  const isSyncRunning = syncStatus.State === 'running';
  const showInitialSpinner = loading || (isSyncRunning && games.length < 1);
  const showEmptyState = !showInitialSpinner && games.length === 0;

  return (
    <DialogBody style={styles.wrapper}>
      <style>{libraryStyles}</style>
      {isSyncRunning && (
        <div className='sentinel-library-sync' aria-live='polite' aria-busy='true'>
          <div className='sentinel-library-sync-meta'>
            <span>Fetching metadata</span>
            <span>
              {syncStatus.Current}/{syncStatus.Total}
            </span>
          </div>
        </div>
      )}
      {showInitialSpinner ? (
        <div className='sentinel-library-loader' aria-busy='true'>
          <Spinner />
        </div>
      ) : showEmptyState ? (
        <EmptyState
          variant='library'
          label='No games found'
          buttonText='Go to Settings'
          buttonClick={() => Navigation.Navigate('/sentinel/settings')}
        />
      ) : (
        <>
          <DialogHeader className={'sentinel-library-header'}>Games</DialogHeader>
          <Focusable className='sentinel-library-grid'>
            {games.map((game) => {
              const progress = computeProgress(game.Achievement.List);
              const isRefreshing = refreshingGameIds.includes(game.AppID);
              return (
                <LibraryImage
                  key={game.AppID}
                  src={game.PortraitImage}
                  fallbackSrc={game.FallbackPortraitImage}
                  alt={game.Name}
                  name={game.Name}
                  progress={progress}
                  isRefreshing={isRefreshing}
                  onActivate={() => Navigation.Navigate(`/sentinel/games/${game.AppID}`)}
                  onOpenContextMenu={(parent) => openGameContextMenu(game.AppID, parent)}
                />
              );
            })}
          </Focusable>
        </>
      )}
    </DialogBody>
  );
};

export default LibraryPage;
