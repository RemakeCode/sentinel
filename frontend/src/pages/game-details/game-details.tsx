import React, { useMemo, useState } from 'react';
import { Link, useLocation, useParams } from 'react-router';
import { ArrowDown, ArrowLeft, ArrowUp, Clock, EyeOff, Ghost, Glasses, History, ListCheck, Trophy } from 'lucide-react';
import { GameBasics } from '@wa/sentinel/backend/steam';
import './game-details.scss';
import { Header } from '@/shared/components/Header/Header';
import { computeProgress } from '@/shared/utils';

interface AchievementProgress {
  earned?: boolean;
  earned_time?: number;
  max_progress?: number;
  progress?: number;
}

type SortOption = 'name-asc' | 'name-desc' | 'time-newest' | 'time-oldest';

const SORT_OPTIONS: { value: SortOption; icon: React.ReactNode; active: SortOption }[] = [
  { value: 'name-asc', icon: <ArrowUp size={20} />, active: 'name-asc' },
  { value: 'name-desc', icon: <ArrowDown size={20} />, active: 'name-desc' },
  { value: 'time-newest', icon: <Clock size={20} />, active: 'time-newest' },
  { value: 'time-oldest', icon: <History size={20} />, active: 'time-oldest' }
];

const STORAGE_KEY = 'game-details-sort';

const GameDetails: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const game = location.state?.game as GameBasics | undefined;

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

  const sortedAchievements = useMemo(() => {
    const list = [...(game?.Achievement.List || [])];
    return list.sort((a, b) => {
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
  }, [game?.Achievement.List, sortBy]);

  const formatUnlockTime = (timestamp: number | undefined): string => {
    if (!timestamp) return '';
    return new Date(timestamp * 1000).toLocaleString();
  };

  return (
    <main className='full-layout'>
      <Header className={'game-details-header'}>
        <Link to='/' viewTransition>
          <ArrowLeft />
        </Link>
        <h2>Achievements</h2>
      </Header>
      <section className='page-content'>
        <div className='game-details-section'>
          <div className='game-details-container'>
            <div className='game-details-container-inner'>
              <div
                className='game-details-image card'
                style={{ viewTransitionName: `game-image-${id}` } as React.CSSProperties}
              >
                <img src={game?.PortraitImage} alt={game?.Name} />
              </div>
              <progress value={stats.percentage} max={100} className='mt-6'></progress>
              <div className='game-details-stats'>
                <div className='game-details-stat-card card'>
                  <Trophy className='game-details-stat-icon' />
                  <span className='game-details-stat-value'>{stats.percentage}%</span>
                  <span className='game-details-stat-label'>Complete</span>
                </div>
                <div className='game-details-stat-card card'>
                  <ListCheck className='game-details-stat-icon' />
                  <span className='game-details-stat-value'>
                    {stats.achievedCount} / {stats.totalCount}
                  </span>
                  <span className='game-details-stat-label'>Total</span>
                </div>
                <div className='game-details-stat-card card'>
                  <Ghost className='game-details-stat-icon' />
                  <span className='game-details-stat-value'>
                    {stats.hiddenEarned} / {stats.hiddenTotal}
                  </span>
                  <span className='game-details-stat-label'>Hidden</span>
                </div>
                <div className='game-details-stat-card card'>
                  <Glasses className='game-details-stat-icon' />
                  <span className='game-details-stat-value'>
                    {stats.visibleEarned} / {stats.visibleTotal}
                  </span>
                  <span className='game-details-stat-label'>Visible</span>
                </div>
              </div>
            </div>
          </div>
          <div className='game-details-ach'>
            <div className='game-details-ach-header'>
              <h1>{game?.Name}</h1>
              <div className='game-details-ach-sort'>
                {SORT_OPTIONS.map((opt) => (
                  <div
                    role='button'
                    key={opt.value}
                    className={`${sortBy === opt.value ? 'active' : ''}`}
                    onClick={() => handleSortChange(opt.value)}
                    title={opt.value.replace('-', ' ')}
                  >
                    {opt.icon}
                  </div>
                ))}
              </div>
            </div>
            <ul className='game-details-ach-list'>
              {sortedAchievements.map((ach, i) => {
                const currentAch = (ach as any).CurrentAch;
                const hasProgress = (currentAch?.max_progress || 0) > 1;
                const progress = currentAch?.progress || 0;
                const maxProgress = currentAch?.max_progress || 1;

                return (
                  <li key={i} className='game-details-ach-item'>
                    <div className='game-details-ach-icon'>
                      <img src={ach.Icon} alt={ach.DisplayName} width={64} height={64} />
                    </div>
                    <div className='game-details-ach-info'>
                      <span className='game-details-ach-title'>{ach.DisplayName}</span>
                      <span className='game-details-ach-desc'>
                        <span className={`${ach.Hidden === 1 ? 'blur' : ''}`}>{ach.Description || ''}</span>
                        {ach.Hidden === 1 && <EyeOff width={18} height={18} />}
                      </span>
                      {hasProgress && (
                        <div className='game-details-ach-progress'>
                          <progress value={progress} max={maxProgress} />
                          <span className='game-details-ach-progress-text'>
                            {progress} / {maxProgress}
                          </span>
                        </div>
                      )}
                    </div>
                    <div className='game-details-ach-unlocktime'>
                      {currentAch?.earned_time ? formatUnlockTime(currentAch.earned_time) : 'Not Unlocked'}
                    </div>
                  </li>
                );
              })}
            </ul>
          </div>
        </div>
      </section>
    </main>
  );
};

export default GameDetails;
