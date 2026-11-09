import './game-details.scss';
import type { FC, ReactNode } from 'react';
import { useMemo, useState } from 'react';
import { motion } from 'framer-motion';
import { Link, useLocation } from 'react-router';
import { ArrowDown, ArrowLeft, ArrowUp, Clock, History, Lock, Trophy } from 'lucide-react';
import type { GameBasics } from '@wa/sentinel/backend/steam';
import { computeProgress } from '@/shared/utils';
import missingCover from '@/assets/images/missing-cover.png';
import { HeaderPortal } from '@/shared/components/header/header';
import { AchievementListItem } from '@/pages/game-details/achievement-list-item';

type SortOption = 'name-asc' | 'name-desc' | 'time-newest' | 'time-oldest';

const SORT_OPTIONS: { value: SortOption; icon: ReactNode; active: SortOption }[] = [
  { value: 'name-asc', icon: <ArrowUp size={20} />, active: 'name-asc' },
  { value: 'name-desc', icon: <ArrowDown size={20} />, active: 'name-desc' },
  { value: 'time-newest', icon: <Clock size={20} />, active: 'time-newest' },
  { value: 'time-oldest', icon: <History size={20} />, active: 'time-oldest' }
];

const STORAGE_KEY = 'game-details-sort';

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.05
    }
  }
};

const GameDetails: FC = () => {
  const location = useLocation();
  // Keep this route's game while AnimatePresence finishes its exit after navigation.
  const [game] = useState<GameBasics | undefined>(() => location.state?.game);

  const [sortBy, setSortBy] = useState<SortOption>(() => {
    if (typeof window !== 'undefined') {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored && SORT_OPTIONS.some((opt) => opt.value === stored)) {
        return stored as SortOption;
      }
    }
    return 'name-asc';
  });

  const handleSortChange = (option: SortOption) => {
    if (option === sortBy) {
      const opposite: Record<SortOption, SortOption> = {
        'name-asc': 'name-desc',
        'name-desc': 'name-asc',
        'time-newest': 'time-oldest',
        'time-oldest': 'time-newest'
      };
      setSortBy(opposite[option]);
      localStorage.setItem(STORAGE_KEY, opposite[option]);
    } else {
      setSortBy(option);
      localStorage.setItem(STORAGE_KEY, option);
    }
  };

  const stats = useMemo(() => {
    const list = game?.Achievement.List || [];
    const earned = list.filter((a) => (a as any).CurrentAch?.earned).length;
    const hidden = list.filter((a) => a.Hidden === 1);
    const hiddenEarned = hidden.filter((a) => (a as any).CurrentAch?.earned).length;
    const visible = list.filter((a) => a.Hidden !== 1);
    const visibleEarned = visible.filter((a) => (a as any).CurrentAch?.earned).length;
    return {
      percentage: computeProgress(game?.Achievement.List),
      achievedCount: earned,
      totalCount: list.length,
      hiddenEarned,
      hiddenTotal: hidden.length,
      visibleEarned,
      visibleTotal: visible.length
    };
  }, [game?.Achievement.List]);

  const { sortedUnlocked, sortedLocked } = useMemo(() => {
    const list = [...(game?.Achievement.List || [])];
    const sorted = list.sort((a, b) => {
      const aTime = (a as any).CurrentAch?.earned_time || 0;
      const bTime = (b as any).CurrentAch?.earned_time || 0;
      switch (sortBy) {
        case 'name-asc':
          return (a.DisplayName || '').localeCompare(b.DisplayName || '');
        case 'name-desc':
          return (b.DisplayName || '').localeCompare(a.DisplayName || '');
        case 'time-newest':
          return bTime - aTime;
        case 'time-oldest':
          return aTime - bTime;
        default:
          return 0;
      }
    });
    const unlocked = sorted.filter((a) => (a as any).CurrentAch?.earned);
    const locked = sorted.filter((a) => !(a as any).CurrentAch?.earned);
    return { sortedUnlocked: unlocked, sortedLocked: locked };
  }, [game?.Achievement.List, sortBy]);

  return (
    <main className='full-layout'>
      <HeaderPortal>
        <div className='header-nav'>
          <Link to='/' className='header-back' aria-label='Back to library'>
            <ArrowLeft />
          </Link>
          <h3>Achievements</h3>
        </div>
      </HeaderPortal>
      <section className='page-content'>
        <div className='desktop-page-intro'>
          <span className='desktop-page-intro-eyebrow'>Game progress</span>
          <h1>{game?.Name}</h1>
          <p>Browse what you've unlocked and see what's left to discover.</p>
        </div>
        <div className='game-details-section'>
          <div className='game-details-container'>
            <div className='game-details-container-inner'>
              <div className='game-details-image card'>
                <img
                  src={game?.PortraitImage}
                  alt={game?.Name}
                  onError={(e) => {
                    e.currentTarget.src = missingCover;
                  }}
                />
              </div>
              <div className='game-details-stats'>
                <div className='game-details-stat-summary'>
                  <div className='game-details-stat-topline'>
                    <strong>{stats.percentage}%</strong>
                    <span>Complete</span>
                  </div>
                  <progress value={stats.percentage} max={100} aria-label='Game completion' />
                </div>
                <div className='game-details-stat-row'>
                  <span>Unlocked</span>
                  <span className='game-details-stat-value'>
                    {stats.achievedCount} / {stats.totalCount}
                  </span>
                </div>
                <div className='game-details-stat-row'>
                  <span>Hidden</span>
                  <span className='game-details-stat-value'>
                    {stats.hiddenEarned} / {stats.hiddenTotal}
                  </span>
                </div>
                <div className='game-details-stat-row'>
                  <span>Visible</span>
                  <span className='game-details-stat-value'>
                    {stats.visibleEarned} / {stats.visibleTotal}
                  </span>
                </div>
              </div>
            </div>
          </div>
          <div className='game-details-ach'>
            <div className='game-details-ach-header'>
              <h2>Achievements</h2>
              <div className='game-details-ach-sort' role='group' aria-label='Sort achievements'>
                {SORT_OPTIONS.map((opt) => (
                  <button
                    type='button'
                    key={opt.value}
                    className={`${sortBy === opt.value ? 'active' : ''}`}
                    onClick={() => handleSortChange(opt.value)}
                    title={opt.value.replace('-', ' ')}
                    aria-label={opt.value.replace('-', ' ')}
                    aria-pressed={sortBy === opt.value}
                  >
                    {opt.icon}
                  </button>
                ))}
              </div>
            </div>
            {sortedUnlocked.length > 0 && (
              <section className='game-details-ach-group' aria-labelledby='unlocked-heading'>
                <h3 className='game-details-ach-subheader' id='unlocked-heading'>
                  <Trophy aria-hidden='true' />
                  Unlocked
                </h3>
                <motion.ul
                  className='game-details-ach-list'
                  variants={containerVariants}
                  initial='hidden'
                  animate='visible'
                >
                  {sortedUnlocked.map((ach, i) => (
                    <AchievementListItem key={`${ach.Name}#${i}`} ach={ach} />
                  ))}
                </motion.ul>
              </section>
            )}
            {sortedLocked.length > 0 && (
              <section className='game-details-ach-group' aria-labelledby='locked-heading'>
                <h3 className='game-details-ach-subheader' id='locked-heading'>
                  <Lock aria-hidden='true' />
                  Locked
                </h3>
                <motion.ul
                  className='game-details-ach-list'
                  variants={containerVariants}
                  initial='hidden'
                  animate='visible'
                >
                  {sortedLocked.map((ach, i) => (
                    <AchievementListItem key={`${ach.Name}#${i}`} ach={ach} />
                  ))}
                </motion.ul>
              </section>
            )}
          </div>
        </div>
      </section>
    </main>
  );
};

export default GameDetails;
