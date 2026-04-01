import { createContext, FC, ReactNode, useContext, useEffect, useState } from 'react';
import { GameBasics } from '@wa/sentinel/backend/steam';
import { Events } from '@wailsio/runtime';
import { LoadAllCachedGameData } from '@wa/sentinel/backend/steam/service';

interface GamesContextType {
  games: (GameBasics | null)[];
  loading: boolean;
  status: number;
  refresh: () => Promise<void>;
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

  const refresh = async () => {
    setLoading(true);
    const data = await LoadAllCachedGameData();
    setGames(data);
    setLoading(false);
  };

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

  return <GamesContext.Provider value={{ games, loading, status, refresh }}>{children}</GamesContext.Provider>;
};
