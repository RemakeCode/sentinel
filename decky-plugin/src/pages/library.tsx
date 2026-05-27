import type { CSSProperties, FC } from 'react';
import { useEffect, useState } from 'react';
import { DialogBody, Focusable, Navigation, PanelSection } from '@decky/ui';
import LibraryImage from '@/shared/components/library-image';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import type { AchievementInfo, GameBasics } from '@/shared/types/GameBasics';

const styles: Record<string, CSSProperties> = {
  wrapper: {
    marginBlock: 'calc(var(--basicui-header-height) + 40px) calc(var(--gamepadui-current-footer-height) + 16px)',
    marginInline: '24px'
  },
  container: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))',
    gap: '24px'
  }
};

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
        setGames([...games, ...data]);
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
      {loading ? (
        <div style={styles.container}>
          {Array.from({ length: 12 }).map((_, i) => (
            <div key={i} style={{ aspectRatio: '2/3', background: 'rgba(255,255,255,0.05)', borderRadius: '4px' }} />
          ))}
        </div>
      ) : games.length === 0 ? (
        <div style={{ textAlign: 'center', color: '#8b929a', padding: '48px 16px' }}>No games found</div>
      ) : (
        <PanelSection title={'Games'}>
          <Focusable style={styles.container}>
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
