import { FC, useEffect, useState } from 'react';
import { DialogBody, DialogButton, DialogHeader, DialogLabel, Focusable, Navigation } from '@decky/ui';
import { LibraryImage } from '@/shared/components/library-image';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import { computeProgress } from '@/shared/utils/utils';
import type { GameBasics } from '@/shared/types/GameBasics';
import { styles } from '@/shared/styles';
import { PiGameController } from 'react-icons/pi';

//language=css
const libraryStyles = `
  .sentinel-library-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 24px;
  }

  .sentinel-library-empty {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    gap: 4px;
    height: 100%
  }

  .sentinel-library-empty-label {
    font-size: 32px;
    font-weight: 700;
  }

  .sentinel-library-header {
    font-size: 24px;
  }
  .sentinel-library-empty-icon {
    width: 80px;
    height: 80px;
    fill: #acb2b8;
    animation: sentinel-wobble 1s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .sentinel-library-empty-button {
    margin-block-start: 16px;
  }
  

  @keyframes sentinel-wobble {
    0%, 100% {
      transform: rotate(0deg);
    }
    25% {
      transform: rotate(15deg);
    }
    50% {
      transform: rotate(0deg);
    }
    75% {
      transform: rotate(-15deg);
    }
  }


`;

const fetcher = new Fetcher();

const LibraryPage: FC = () => {
  const [games, setGames] = useState<GameBasics[]>([]);
  const [loading, setLoading] = useState(true);

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
        <div className='sentinel-library-empty'>
          <DialogLabel className='sentinel-library-empty-label'>No games found</DialogLabel>
          <PiGameController className='sentinel-library-empty-icon' />

          <DialogButton
            className='sentinel-library-empty-button'
            onClick={() => Navigation.Navigate('/sentinel/settings')}
          >
            Go to Settings
          </DialogButton>
        </div>
      ) : (
        <>
          <DialogHeader className={'sentinel-library-header'}>Games</DialogHeader>
          <Focusable className='sentinel-library-grid'>
            {games.map((game) => {
              const progress = computeProgress(game.Achievement.List);
              return (
                <LibraryImage
                  key={game.AppID}
                  src={game.PortraitImage}
                  alt={game.Name}
                  name={game.Name}
                  progress={progress}
                  onActivate={() => Navigation.Navigate(`/sentinel/games/${game.AppID}`)}
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
