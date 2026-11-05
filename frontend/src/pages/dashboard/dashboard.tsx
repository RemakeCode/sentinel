import './dashboard.scss';
import type { CSSProperties, FC } from 'react';
import { motion, type Variants } from 'framer-motion';

import { Gamepad2, Settings } from 'lucide-react';
import { Link } from 'react-router';
import EmptyState from '@/shared/components/empty-state';
import { computeProgress } from '@/shared/utils';
import { useGames } from '@/shared/context/games-context';
import logo from '@/assets/images/sentinel.webp';
import missingCover from '@/assets/images/missing-cover.png';
import { HeaderPortal } from '@/shared/components/header/header';

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.03
    }
  }
};

const itemVariants: Variants = {
  hidden: { opacity: 0, y: 10, scale: 0.98 },
  visible: {
    opacity: 1,
    y: 0,
    scale: 1,
    transition: {
      duration: 0.3,
      ease: 'easeOut'
    }
  }
};

const Dashboard: FC = () => {
  const { games, loading, syncStatus, isRefreshingGame } = useGames();
  const showLoader = loading || (syncStatus.State === 'running' && games.length === 0);

  return (
    <main className='full-layout'>
      <HeaderPortal>
        <div className='logo'>
          <img src={logo} alt='sentinel logo' width='50px' height='50px' />
          <div className='logo-text'>
            <h3 className='logo-text-label'>Sentinel</h3>
            <div className='logo-text-meta'>An achievement watcher</div>
          </div>
        </div>
        <Link to='/settings' className='header-icon' aria-label='Settings'>
          <Settings />
        </Link>
      </HeaderPortal>
      <section className='page-content'>
        <div className='desktop-page-intro'>
          <span className='desktop-page-intro-eyebrow'>Your collection</span>
          <h1>Library</h1>
          <p>See your progress, pick up where you left off, and explore achievements.</p>
        </div>
        <div className='dashboard-section-header'>
          <Gamepad2 aria-hidden='true' />
          <h2>Games</h2>
          <span className='dashboard-game-count' aria-label={`${games.length} games`}>
            {games.length}
          </span>
        </div>

        {showLoader ? (
          <div className='dashboard-loader' aria-busy='true' data-spinner='large' />
        ) : games.length === 0 ? (
          <div className='dashboard-empty-state'>
            <EmptyState
              message={syncStatus.State === 'error' ? 'Game metadata could not be loaded.' : 'No games found.'}
              icon={<Gamepad2 />}
            />
          </div>
        ) : (
          <motion.div className='games-container' variants={containerVariants} initial='hidden' animate='visible'>
            {games.map((game, idx) => {
              if (!game) {
                return null;
              }

              const progress = computeProgress(game?.Achievement.List);
              const isRefreshing = isRefreshingGame(game.AppID);

              return (
                <motion.div key={game.AppID || `${game.Name}#${idx}`} variants={itemVariants} className='games-item'>
                  <div
                    className={`games-item-shell ${isRefreshing ? 'is-refreshing' : ''}`}
                    style={
                      {
                        '--custom-contextmenu': 'game-card-menu',
                        '--custom-contextmenu-data': JSON.stringify({ appId: game.AppID })
                      } as CSSProperties
                    }
                  >
                    <Link
                      to={`/game/${game.AppID}`}
                      state={{ game, idx }}
                      className='games-item-link'
                      aria-label={`Open ${game.Name} achievements, ${progress}% complete`}
                    >
                      <div className='games-item-card card'>
                        <div className='games-item-cover'>
                          <img
                            src={game.PortraitImage}
                            alt={game.Name || ''}
                            onError={(e) => {
                              e.currentTarget.src = missingCover;
                            }}
                          />
                        </div>
                        <div className='games-item-content'>
                          <div className='games-item-title'>{game.Name}</div>
                          <div className='games-item-meta'>
                            {progress}%
                          </div>
                          <progress value={progress} max={100} aria-label={`${game.Name} completion`} />
                        </div>
                      </div>
                    </Link>
                    {isRefreshing && (
                      <div className='games-item-refreshing' aria-live='polite' aria-busy='true' data-spinner='small'>
                        <span>Refreshing...</span>
                      </div>
                    )}
                  </div>
                </motion.div>
              );
            })}
          </motion.div>
        )}
      </section>
    </main>
  );
};

export default Dashboard;
