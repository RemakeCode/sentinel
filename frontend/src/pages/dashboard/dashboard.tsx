import './dashboard.scss';
import React, { useEffect, useState } from 'react';

import { Search, Settings } from 'lucide-react';
import { Link } from 'react-router';
import EmptyState from '@/shared/components/EmptyState';
import { LoadAllCachedGameData } from '@wa/sentinel/backend/steam/service';
import { Events } from '@wailsio/runtime';
import { GameBasics } from '@wa/sentinel/backend/steam';
import { Header } from '@/shared/components/Header/Header';
import { Achievement } from '@wa/sentinel/backend/ach';

// Cache globally so returning to Dashboard doesn't trigger Skeletons and break view transitions
let globalCachedGames: (GameBasics | null)[] | null = null;

/**
 * Computes the progress percentage based on earned achievements
 * @param currentAch Object containing achievements with their earned status
 * @returns Progress percentage rounded to two decimal places
 */
const computeProgress = (currentAch: { [p: string]: Achievement | undefined } | undefined): number => {
  if (!currentAch) return 0;

  const achievements = Object.values(currentAch);
  if (achievements.length === 0) return 0;

  const earnedCount = achievements.filter((ach) => ach?.earned).length;

  return Math.round((earnedCount / achievements.length) * 100);
};

const Dashboard: React.FC = () => {
  const [games, setGames] = useState<(GameBasics | null)[]>(globalCachedGames || []);
  const [loading, setLoading] = useState(!globalCachedGames);
  const [status, setStatus] = useState<number>(0);

  useEffect(() => {
    Events.On('sentinel::fetch-status', (event) => {
      const percentage = Math.floor((event.data.Current / event.data.Total) * 100);
      setStatus(percentage);
    });
    return () => Events.Off('sentinel::fetch-status');
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
      }
    };
    handleGames();
  }, []);

  //TODO: Only show the page when status is at 100%.
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

        {/*<div>{status}%</div>*/}
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
        ) : games.length === 0 ? (
          <EmptyState message='No games found. Add an emulator path in settings!' />
        ) : (
          <div className='games-container'>
            {games.map((game, idx) => {
              const progress = computeProgress(game?.CurrentAch);

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
