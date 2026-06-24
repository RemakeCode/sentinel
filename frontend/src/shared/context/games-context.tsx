import type { FC, ReactNode } from 'react';
import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { LibrarySyncStatus, type GameBasics } from '@wa/sentinel/backend/steam';
import { Events } from '@wailsio/runtime';
import { GetLibrarySyncStatus, LoadAllCachedGameData, RefetchGameData } from '@wa/sentinel/backend/steam/service';
import LibrarySyncAlert from '@/shared/components/library-sync-alert';

interface GamesContextType {
  games: (GameBasics | null)[];
  loading: boolean;
  status: number;
  refresh: () => Promise<void>;
  refreshGame: (appID: string) => Promise<void>;
  isRefreshingGame: (appID: string) => boolean;
}

const GamesContext = createContext<GamesContextType | undefined>(undefined);

export const useGames = () => {
  const context = useContext(GamesContext);
  if (!context) {
    throw new Error('useGames must be used within a GamesProvider');
  }
  return context;
};

interface GamesProviderProps {
  children: ReactNode;
}

const SYNC_POLL_INTERVAL_MS = 1000;

const emptySyncStatus = new LibrarySyncStatus({ State: 'idle', Current: 0, Total: 0 });

const getSyncPercentage = (syncStatus: LibrarySyncStatus) => {
  if (syncStatus.Total === 0) {
    return syncStatus.State === 'done' ? 100 : 0;
  }

  const percentage = Math.floor((syncStatus.Current / syncStatus.Total) * 100);
  return syncStatus.State === 'running' ? Math.max(1, percentage) : percentage;
};

export const GamesProvider: FC<GamesProviderProps> = ({ children }) => {
  const [games, setGames] = useState<(GameBasics | null)[]>([]);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<number>(0);
  const [syncStatus, setSyncStatus] = useState<LibrarySyncStatus>(emptySyncStatus);
  const [isInitialized, setIsInitialized] = useState(false);
  const [refreshingGameIDs, setRefreshingGameIDs] = useState<string[]>([]);
  const refreshingGameIDsRef = useRef<Set<string>>(new Set());
  const lastSyncCurrentRef = useRef(0);

  const setRefreshingState = (updater: (current: Set<string>) => Set<string>) => {
    const next = updater(new Set(refreshingGameIDsRef.current));
    refreshingGameIDsRef.current = next;
    setRefreshingGameIDs(Array.from(next));
  };

  const replaceGame = (updatedGame: GameBasics) => {
    setGames((current) =>
      current.map((game) => {
        if (!game || game.AppID !== updatedGame.AppID) {
          return game;
        }

        return updatedGame;
      })
    );
  };

  const loadCachedGames = useCallback(async (showLoading = false) => {
    if (showLoading) {
      setLoading(true);
    }

    try {
      const data = await LoadAllCachedGameData();
      setGames(data);
    } catch (error) {
      console.error(error);
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  }, []);

  const refresh = useCallback(async () => loadCachedGames(true), [loadCachedGames]);

  const refreshGame = async (appID: string) => {
    if (!appID || refreshingGameIDsRef.current.has(appID)) {
      return;
    }

    setRefreshingState((current) => {
      current.add(appID);
      return current;
    });

    try {
      const refreshedGame = await RefetchGameData(appID);

      if (!refreshedGame) {
        throw new Error(`No refreshed game data returned for appID ${appID}`);
      }

      replaceGame(refreshedGame);
      window.ot?.toast(`${refreshedGame.Name || 'Game'} refreshed`, 'Success', { variant: 'success' });
    } catch (error) {
      console.error(`Failed to refresh game ${appID}:`, error);
      window.ot?.toast('Failed to refresh game', 'Error', { variant: 'danger' });
      throw error;
    } finally {
      setRefreshingState((current) => {
        current.delete(appID);
        return current;
      });
    }
  };

  const isRefreshingGame = (appID: string) => refreshingGameIDs.includes(appID);

  const refreshGameRef = useRef(refreshGame);

  useEffect(() => {
    refreshGameRef.current = refreshGame;
  });

  useEffect(() => {
    const unsubscribers: (() => void)[] = [];

    const handleDataUpdated = async () => {
      if (isInitialized) {
        await refresh();
      }
    };

    unsubscribers.push(Events.On('sentinel::data-updated', handleDataUpdated));

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [isInitialized, refresh]);

  useEffect(() => {
    const handleInitialLoad = async () => {
      try {
        const [data, currentSyncStatus] = await Promise.all([
          LoadAllCachedGameData().catch((error) => {
            console.error(error);
            return [];
          }),
          GetLibrarySyncStatus()
        ]);

        setGames(data);
        setSyncStatus(currentSyncStatus);
        setStatus(getSyncPercentage(currentSyncStatus));
        lastSyncCurrentRef.current = currentSyncStatus.Current;
        setIsInitialized(currentSyncStatus.State !== 'running');
      } catch (e) {
        console.error(e);
        setIsInitialized(true);
      } finally {
        setLoading(false);
      }
    };
    handleInitialLoad();
  }, []);

  useEffect(() => {
    if (syncStatus.State !== 'running') {
      return;
    }

    let active = true;

    const pollSyncStatus = async () => {
      try {
        const currentSyncStatus = await GetLibrarySyncStatus();

        if (!active) {
          return;
        }

        const previousCurrent = lastSyncCurrentRef.current;
        setSyncStatus(currentSyncStatus);
        setStatus(getSyncPercentage(currentSyncStatus));

        if (currentSyncStatus.Current > previousCurrent) {
          lastSyncCurrentRef.current = currentSyncStatus.Current;
          await loadCachedGames(false);
        }

        if (currentSyncStatus.State === 'done' || currentSyncStatus.State === 'error') {
          setLoading(false);
          setIsInitialized(true);
        }
      } catch (error) {
        console.error(error);
      }
    };

    void pollSyncStatus();
    const intervalID = window.setInterval(pollSyncStatus, SYNC_POLL_INTERVAL_MS);

    return () => {
      active = false;
      window.clearInterval(intervalID);
    };
  }, [loadCachedGames, syncStatus.State]);

  useEffect(() => {
    const unsubscribe = Events.On('sentinel::refresh-game-requested', (event: { data: string }) => {
      void refreshGameRef.current(event.data);
    });

    return () => unsubscribe();
  }, []);

  return (
    <GamesContext.Provider value={{ games, loading, status, refresh, refreshGame, isRefreshingGame }}>
      <LibrarySyncAlert syncStatus={syncStatus} />
      {children}
    </GamesContext.Provider>
  );
};
