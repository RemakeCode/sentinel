import type { FC, ReactNode } from 'react';
import { createContext, useContext, useEffect, useRef, useState } from 'react';
import { GameBasics } from '@wa/sentinel/backend/steam';
import { Events } from '@wailsio/runtime';
import { LoadAllCachedGameData, RefetchGameData } from '@wa/sentinel/backend/steam/service';

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

export const GamesProvider: FC<GamesProviderProps> = ({ children }) => {
  const [games, setGames] = useState<(GameBasics | null)[]>([]);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<number>(0);
  const [isInitialized, setIsInitialized] = useState(false);
  const [refreshingGameIDs, setRefreshingGameIDs] = useState<string[]>([]);
  const refreshingGameIDsRef = useRef<Set<string>>(new Set());

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

  const refresh = async () => {
    setLoading(true);

    try {
      const data = await LoadAllCachedGameData();
      setGames(data);
    } finally {
      setLoading(false);
    }
  };

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

    const handleFetchStatus = async (event: { data: { Current: number; Total: number } }) => {
      const current = event.data.Current;
      const total = event.data.Total;
      const percentage = Math.floor((current / total) * 100);
      setStatus(percentage);
      setLoading(true);

      if (current === total) {
        await refresh();
        setLoading(false);
        setIsInitialized(true);
      }
    };

    const handleDataUpdated = async () => {
      if (isInitialized) {
        await refresh();
      }
    };

    unsubscribers.push(Events.On('sentinel::fetch-status', handleFetchStatus));
    unsubscribers.push(Events.On('sentinel::data-updated', handleDataUpdated));

    return () => {
      unsubscribers.forEach((unsub) => unsub());
    };
  }, [isInitialized]);

  useEffect(() => {
    const handleInitialLoad = async () => {
      try {
        const data = await LoadAllCachedGameData();
        setGames(data);
        setLoading(false);
        setIsInitialized(true);
      } catch (e) {
        console.error(e);
        setLoading(false);
        setIsInitialized(true);
      }
    };
    handleInitialLoad();
  }, []);

  useEffect(() => {
    const unsubscribe = Events.On('sentinel::refresh-game-requested', (event: { data: string }) => {
      void refreshGameRef.current(event.data);
    });

    return () => unsubscribe();
  }, []);

  return <GamesContext.Provider value={{ games, loading, status, refresh, refreshGame, isRefreshingGame }}>{children}</GamesContext.Provider>;
};
