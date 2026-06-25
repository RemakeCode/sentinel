import { FC, useEffect, useState } from 'react';
import { DialogBody, DialogHeader, Focusable, Menu, MenuItem, Navigation, showContextMenu } from '@decky/ui';
import { toaster } from '@decky/api';
import { LibraryImage } from '@/shared/components/library-image';
import { EmptyState } from '@/shared/components/empty-state';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import { computeProgress } from '@/shared/utils/utils';
import type { GameBasics } from '@/shared/types/GameBasics';
import { styles } from '@/shared/styles';

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
`;

const fetcher = new Fetcher();

const LibraryPage: FC = () => {
  const [games, setGames] = useState<GameBasics[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshingGameIds, setRefreshingGameIds] = useState<string[]>([]);

  useEffect(() => {
    const load = async () => {
      try {
        const data = await fetcher.get<GameBasics[]>(`${BASE_URL}/games`);
        setGames(data);
      } catch {
        setGames([]);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  const handleRefreshGame = async (appId: string) => {
    if (!appId || refreshingGameIds.includes(appId)) {
      return;
    }

    setRefreshingGameIds((current) => [...current, appId]);

    try {
      const refreshedGame = await fetcher.post<GameBasics>(`${BASE_URL}/games/${appId}/refresh`, {});
      setGames((current) =>
        current.map((game) => {
          if (game.AppID !== refreshedGame.AppID) {
            return game;
          }

          return refreshedGame;
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

  return (
    <DialogBody style={styles.wrapper}>
      <style>{libraryStyles}</style>
      {loading ? (
        <div className='sentinel-library-grid'>
          {Array.from({ length: 12 }).map((_, i) => (
            <div key={i} style={{ aspectRatio: '2/3', background: 'rgba(255,255,255,0.05)', borderRadius: '4px' }} />
          ))}
        </div>
      ) : games.length === 0 ? (
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
