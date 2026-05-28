import { FC, useEffect, useState } from 'react';
import { DialogBody, DialogLabel, Focusable, Navigation, PanelSection } from '@decky/ui';
import { LibraryImage } from '@/shared/components/library-image';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import type { AchievementInfo, GameBasics } from '@/shared/types/GameBasics';
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
  }

  @keyframes sentinel-wobble {
    0%, 100% { transform: rotate(0deg); }
    25% { transform: rotate(15deg); }
    50% { transform: rotate(0deg); }
    75% { transform: rotate(-15deg); }
  }
  
  .sentinel-library-empty-icon {
    width: 80px;
    height: 80px;
    fill: #acb2b8;
    animation: sentinel-wobble 1s ease-in-out 1;
  }
`;

const fetcher = new Fetcher();

function computeProgress(list: AchievementInfo[]): number {
  if (!list || list.length === 0) return 0;
  const earned = list.filter((a) => a.CurrentAch?.earned).length;
  return Math.round((earned / list.length) * 100);
}

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
        </div>
      ) : (
        <PanelSection title={'Games'}>
          <Focusable className='library-grid'>
            {games.map((game) => {
              const progress = computeProgress(game.Achievement.List);
              return (
                <LibraryImage
                  key={game.AppID}
                  src={game.PortraitImage}
                  alt={game.Name}
                  name={game.Name}
                  progress={progress}
                  onActivate={() => Navigation.Navigate(`/sentinel/game/${game.AppID}`)}
                />
              );
            })}
          </Focusable>
        </PanelSection>
      )}
    </DialogBody>
  );
};

export default LibraryPage;
