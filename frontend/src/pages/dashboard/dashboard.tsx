import './dashboard.scss';
import type { CSSProperties, FC } from 'react';

import { Gamepad2, Settings } from 'lucide-react';
import { Link } from 'react-router';
import EmptyState from '@/shared/components/empty-state';
import { Header } from '@/shared/components/header/header';
import { computeProgress } from '@/shared/utils';
import { useGames } from '@/shared/context/games-context';
import logo from '@/assets/images/sentinel.webp';

const Dashboard: FC = () => {
  const { games, loading, status } = useGames();

  return (
    <main className='full-layout'>
      <Header className='dashboard-header'>
        <div className='logo'>
          <img src={logo} alt='sentinel logo' width='50px' height='50px' />
          <div className='logo-text'>
            <div className='logo-text-label'>Sentinel</div>
            <div className='logo-text-meta'>An achievement watcher</div>
          </div>
        </div>
        {/*<div className='dashboard-header-search-bar'>*/}
        {/*  <fieldset className='group'>*/}
        {/*    <input type='text' placeholder='search...' />*/}
        {/*    <button className='outline'>*/}
        {/*      <Search />*/}
        {/*    </button>*/}
        {/*  </fieldset>*/}
        {/*</div>*/}

        <Link to='/settings' viewTransition className='dashboard-header-settings-link'>
          <Settings className='dashboard-header-settings-link-icon' />
        </Link>
      </Header>
      <section className='page-content'>
        <h2 className='dashboard-section-header'>Library</h2>

        {loading ? (
          <div className='dashboard-loader'>
            {Array(100)
              .fill(1)
              .map((_, i) => (
                <div role='status' className='skeleton box' key={i} />
              ))}
          </div>
        ) : games.length === 0 && status === 0 ? (
          <div className='dashboard-empty-state'>
            <EmptyState message='No games found.' icon={<Gamepad2 />} />
          </div>
        ) : (
          <div className='games-container'>
            {games.map((game, idx) => {
              const progress = computeProgress(game?.Achievement.List);

              return (
                <Link
                  to={`/game/${game?.AppID}`}
                  state={{ game, idx }}
                  viewTransition
                  className='games-item'
                  key={`${game?.Name}#${idx}`}
                >
                  <div
                    className='games-item-card card'
                    style={{ viewTransitionName: `game-image-${idx}` } as CSSProperties}
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
