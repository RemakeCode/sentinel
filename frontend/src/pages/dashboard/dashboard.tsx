import './dashboard.scss';
import React, { useEffect, useState } from 'react';

import { Search, Settings } from 'lucide-react';
import { Link } from 'react-router';
import EmptyState from '@/shared/components/EmptyState';
import { LoadAllCachedGameData } from '@wa/sentinel/backend/steam/service';
import { Events } from '@wailsio/runtime';
import { GameBasics } from '@wa/sentinel/backend/steam';
import { Header } from '@/shared/components/Header/Header';
import { computeProgress } from '@/shared/utils';

// Cache globally so returning to Dashboard doesn't trigger Skeletons and break view transitions
let globalCachedGames: (GameBasics | null)[] | null = null;

const Dashboard: React.FC = () => {
  const [games, setGames] = useState<(GameBasics | null)[]>(globalCachedGames || []);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<number>(0);

  useEffect(() => {
    const unsubscribe = Events.On('sentinel::fetch-status', async (event) => {
      const current = event.data.Current;
      const total = event.data.Total;
      const percentage = Math.floor((current / total) * 100);
      setStatus(percentage);
      setLoading(true);

      if (current === total) {
        const data = await LoadAllCachedGameData();
        globalCachedGames = data;
        setGames(data);
        setLoading(false);
      }
    });

    return () => unsubscribe();
  }, []);

  useEffect(() => {
    if (globalCachedGames) return;
    const handleGames = async () => {
      try {
        const data = await LoadAllCachedGameData();
        globalCachedGames = data;
        setGames(data);
        setLoading(false);
      } catch (e) {
        console.error(e);
        setLoading(false);
      }
    };
    handleGames();
  }, []);

  console.log({ loading, status });
  return (
    <main className='full-layout'>
      <Header>
        <div className='dashboard-header-search-bar'>
          <fieldset className='group'>
            <input type='text' placeholder='search...' />
            <button className='outline'>
              <Search />
            </button>
          </fieldset>
        </div>

        <Link to='/settings' viewTransition className='dashboard-header-settings-link'>
          <Settings className='dashboard-header-settings-link-icon' />
        </Link>
      </Header>
      <section className='main-content'>
        <div className='dashboard-section-header'>
          <h2 className='dashboard-section-header-label'>Library</h2>
          <div className='dashboard-section-header-actions'>View All</div>
        </div>

        {loading ? (
          <div className='dashboard-loader'>
            {Array(100)
              .fill(1)
              .map((_, i) => (
                <div role='status' className='skeleton box' key={i} />
              ))}
          </div>
        ) : games.length === 0 && status === 0 ? (
          <EmptyState message='No games found.' />
        ) : (
          <div className='games-container'>
            {games.map((game, idx) => {
              const progress = computeProgress(game?.Achievement.List);

              return (
                <Link
                  to={`/game/${idx}`}
                  state={{ game }}
                  viewTransition
                  className='games-item'
                  key={`${game?.Name}#${idx}`}
                >
                  <div
                    className='games-item-card card'
                    style={{ viewTransitionName: `game-image-${idx}` } as React.CSSProperties}
                  >
                    <div className='games-item-progress'>
                      <progress value={progress} max={100} />
                    </div>
                    <img src={game?.PortraitImage} alt={game?.Name || ''} />

                    <div className='games-item-overlay'>
                      <div className='games-item-title'>{game?.Name}</div>
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        )}
      </section>
    </main>
  );
};

export default Dashboard;
