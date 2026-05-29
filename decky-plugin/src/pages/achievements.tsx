import { FC, ReactNode, useEffect, useMemo, useState } from 'react';
import {
  achievementListClasses,
  achievementPageClasses,
  DialogBody,
  DialogBodyText,
  DialogHeader,
  Focusable,
  joinClassNames,
  Marquee,
  ProgressBar,
  ProgressBarWithInfo,
  ScrollPanel,
  staticClasses
} from '@decky/ui';

import { LibraryImage } from '@/shared/components/library-image';
import { BASE_URL, Fetcher, IMG_URL } from '@/shared/utils/fetcher';
import type { AchievementInfo, GameBasics } from '@/shared/types/GameBasics';
import { styles } from '@/shared/styles';
import { FaArrowDown, FaArrowUp, FaClock, FaHistory } from 'react-icons/fa';

const fetcher = new Fetcher();

type SortOption = 'name-asc' | 'name-desc' | 'time-newest' | 'time-oldest';

const SORT_OPTIONS: { value: SortOption; icon: ReactNode }[] = [
  { value: 'name-asc', icon: <FaArrowUp size={20} /> },
  { value: 'name-desc', icon: <FaArrowDown size={20} /> },
  { value: 'time-newest', icon: <FaClock size={20} /> },
  { value: 'time-oldest', icon: <FaHistory size={20} /> }
];

function computeProgress(list: AchievementInfo[]): number {
  if (!list || list.length === 0) return 0;
  const earned = list.filter((a) => a.CurrentAch?.earned).length;
  return Math.round((earned / list.length) * 100);
}

function formatUnlockTime(timestamp: number | undefined): string {
  if (!timestamp) return 'Locked';
  const tsSeconds = timestamp > Math.floor(Date.now() / 1000) ? Math.floor(timestamp / 1000) : timestamp;
  return new Date(tsSeconds * 1000).toLocaleString();
}

//language=css
const achievementStyles = `
  .sentinel-achievement-container {
    display: grid;
    grid-template-columns: minmax(200px, 300px) 1fr;
    align-items: start;
    gap: 8px;
  }

  .sentinel-achievement-sidebar {
    position: sticky;
    top: 0;
    height: 100%;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .sentinel-achievement-sidebar-image {
    display: flex;
    padding: 0;
    width: 100%;
    margin-inline-end: 16px;
  }

  .sentinel-achievement-stats {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
    animation-name: fadeIn;
    animation-timing-function: ease-in;
    animation-duration: 300ms;
    margin-block-start: 8px
  }

  .sentinel-achievement-stats-card {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    padding: 12px;
    text-align: center;
    flex: 1;
  }

  .sentinel-achievement-stat-value {
    font-size: 18px;
    font-weight: bold;
    display: block;
  }

  .sentinel-achievement-stat-label {
    font-size: 12px;
    color: #8b929a;
    display: block;
    margin-block-start: 2px;
  }

  .sentinel-achievement-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    position: sticky;
    top: 0;
    z-index: 5;
    padding: 16px;
    width: 100%;
    background: #000; 
    box-sizing: border-box;
    border-radius: 4px;
    font-size: 24px;
  }

  .sentinel-achievement-sort-buttons {
    display: flex;
    gap: 16px;
  }

  .sentinel-achievement-sort-button {
    width: 30px;
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
  }

  .sentinel-achievement-sort-button--active, .sentinel-achievement-sort-button--focus {
    background: hsla(0, 0%, 100%, .5);
  }


  .sentinel-achievement-subheader {
    position: sticky;
    top: 60px;
    background: inherit;
    width: 100%;
  }

  .sentinel-achievement-item {
    display: flex;
    flex-direction: row;
    align-items: center;
    padding-inline: 8px;
    gap: 8px;
  }

  .sentinel-achievement-icon {
    width: 60px;
    height: 60px;
    border-radius: 6px;
  }

  .sentinel-achievement-meta {
    flex: 1;
  }

  .sentinel-achievement-state {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

`;

const AchievementsPage: FC = () => {
  const appId = window.location.pathname.split('/games/')[1];

  const [game, setGame] = useState<GameBasics | null>(null);
  const [globalPercentages, setGlobalPercentages] = useState<Map<string, number>>(new Map());
  const [isLoading, setIsLoading] = useState(true);
  const [sortBy, setSortBy] = useState<SortOption>('name-asc');

  useEffect(() => {
    const dom = document.querySelectorAll('.sentinel-achievement-section-header');

    console.log({ dom });

    const loadData = async () => {
      if (!appId) return;
      try {
        const games = await fetcher.get<GameBasics[]>(`${BASE_URL}/games`);
        const found = games.find((g) => g.AppID === appId);
        setGame(found ?? null);
      } catch {
        setGame(null);
      }
    };
    loadData();
  }, [appId]);

  useEffect(() => {
    const loadPercentages = async () => {
      if (!appId) return;
      try {
        const achievements = await fetcher.get<Array<{ name: string; percent: string }>>(
          `${BASE_URL}/games/${appId}/global-achievement-percentages`
        );
        const map = new Map<string, number>();
        achievements.forEach((ach) => map.set(ach.name, parseFloat(ach.percent)));
        setGlobalPercentages(map);
      } catch {
        // global percentages unavailable
      } finally {
        setIsLoading(false);
      }
    };
    loadPercentages();
  }, [appId]);

  const stats = useMemo(() => {
    const list = game?.Achievement.List || [];
    const earned = list.filter((a) => a.CurrentAch?.earned).length;
    const hidden = list.filter((a) => a.Hidden === 1);
    const hiddenEarned = hidden.filter((a) => a.CurrentAch?.earned).length;
    const visible = list.filter((a) => a.Hidden !== 1);
    const visibleEarned = visible.filter((a) => a.CurrentAch?.earned).length;
    return {
      percentage: computeProgress(list),
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
      const aTime = a.CurrentAch?.earned_time || 0;
      const bTime = b.CurrentAch?.earned_time || 0;
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
    return {
      sortedUnlocked: sorted.filter((a) => a.CurrentAch?.earned),
      sortedLocked: sorted.filter((a) => !a.CurrentAch?.earned)
    };
  }, [game?.Achievement.List, sortBy]);

  const handleSortChange = (option: SortOption) => {
    if (option === sortBy) {
      const opposite: Record<SortOption, SortOption> = {
        'name-asc': 'name-desc',
        'name-desc': 'name-asc',
        'time-newest': 'time-oldest',
        'time-oldest': 'time-newest'
      };
      setSortBy(opposite[option]);
    } else {
      setSortBy(option);
    }
  };

  if (!game) {
    return (
      <DialogBody style={styles.wrapper}>
        <DialogBodyText>Unable to locate achievement page</DialogBodyText>
      </DialogBody>
    );
  }

  return (
    <div style={styles.wrapper}>
      <style>{achievementStyles}</style>
      <div className='sentinel-achievement-container'>
        <div className='sentinel-achievement-sidebar'>
          <div className='sentinel-achievement-sidebar-image'>
            <LibraryImage src={game.PortraitImage} alt={game.Name} />
          </div>
          <ProgressBar nProgress={stats.percentage} focusable={false} />
          <div className='sentinel-achievement-stats'>
            <div className='sentinel-achievement-stats-card'>
              <span className='sentinel-achievement-stat-value'>{stats.percentage}%</span>
              <span className='sentinel-achievement-stat-label'>Complete</span>
            </div>
            <div className='sentinel-achievement-stats-card'>
              <span className='sentinel-achievement-stat-value'>
                {stats.achievedCount}/{stats.totalCount}
              </span>
              <span className='sentinel-achievement-stat-label'>Total</span>
            </div>
            <div className='sentinel-achievement-stats-card'>
              <span className='sentinel-achievement-stat-value'>
                {stats.hiddenEarned}/{stats.hiddenTotal}
              </span>
              <span className='sentinel-achievement-stat-label'>Hidden</span>
            </div>
            <div className='sentinel-achievement-stats-card'>
              <span className='sentinel-achievement-stat-value'>
                {stats.visibleEarned}/{stats.visibleTotal}
              </span>
              <span className='sentinel-achievement-stat-label'>Visible</span>
            </div>
          </div>
        </div>
        {/*//@ts-ignore-check*/}
        <ScrollPanel style={styles.achList}>
          <Focusable className={joinClassNames(staticClasses.PanelSectionTitle, 'sentinel-achievement-header')}>
            <Marquee>{game.Name}</Marquee>
            <div className='sentinel-achievement-sort-buttons'>
              {SORT_OPTIONS.map((opt) => (
                <Focusable
                  noFocusRing={true}
                  key={opt.value}
                  focusClassName={'sentinel-achievement-sort-button--focus'}
                  className={joinClassNames(
                    'sentinel-achievement-sort-button',
                    sortBy === opt.value ? 'sentinel-achievement-sort-button--active' : ''
                  )}
                  onActivate={() => handleSortChange(opt.value)}
                  title={opt.value.replace('-', ' ')}
                >
                  {opt.icon}
                </Focusable>
              ))}
            </div>
          </Focusable>
          <div className={achievementPageClasses.AchievementTabs} style={{ height: 'auto' }}>
            <div className={joinClassNames(achievementListClasses.AchievementList)}>
              {sortedUnlocked.length > 0 && (
                <>
                  <DialogHeader
                    className={joinClassNames(staticClasses.PanelSectionTitle, 'sentinel-achievement-subheader')}
                  >
                    Unlocked Achievements({sortedUnlocked.length})
                  </DialogHeader>
                  {sortedUnlocked.map((ach, i) => {
                    const currentAch = ach.CurrentAch;
                    const hasProgress = (currentAch?.max_progress || 0) > 1;
                    const progress = currentAch?.progress || 0;
                    const maxProgress = currentAch?.max_progress || 1;

                    return (
                      <Focusable
                        key={`${ach.Name}#${i}`}
                        className={joinClassNames(
                          achievementListClasses.AchievementListItemBase,
                          'sentinel-achievement-item'
                        )}
                        onActivate={() => {}}
                      >
                        <img
                          src={`${IMG_URL}${ach.Icon}`}
                          alt={ach.DisplayName}
                          className='sentinel-achievement-icon'
                        />

                        <div className='sentinel-achievement-meta'>
                          <div className={achievementListClasses.AchievementTitle}>{ach.DisplayName}</div>
                          <div
                            className={joinClassNames(
                              achievementListClasses.AchievementDescription,
                              ach.Hidden === 1 ? achievementListClasses.Hidden : ''
                            )}
                          >
                            {ach.Description || ''}
                          </div>
                          {!isLoading && globalPercentages.has(ach.Name) && (
                            <div
                              className={joinClassNames(
                                achievementListClasses.AchievementGlobalPercentage,
                                achievementListClasses.InBody
                              )}
                            >
                              {globalPercentages.get(ach.Name)}% of players have this
                            </div>
                          )}
                        </div>

                        <div className='sentinel-achievement-state'>
                          <div className={achievementListClasses.UnlockDate}>
                            Unlocked {formatUnlockTime(currentAch.earned_time)}
                          </div>

                          {hasProgress && (
                            <ProgressBar
                              nProgress={Math.round(
                                ((currentAch?.earned && progress !== maxProgress ? progress + 1 : progress) /
                                  maxProgress) *
                                  100
                              )}
                              focusable={false}
                            />
                          )}
                        </div>
                      </Focusable>
                    );
                  })}
                </>
              )}

              {sortedLocked.length > 0 && (
                <>
                  <DialogHeader className='sentinel-achievement-subheader'>
                    Locked Achievements({sortedLocked.length})
                  </DialogHeader>
                  {sortedLocked.map((ach, i) => {
                    const currentAch = ach.CurrentAch;
                    const hasProgress = (currentAch?.max_progress || 0) > 1;
                    const progress = currentAch?.progress || 0;
                    const maxProgress = currentAch?.max_progress || 1;

                    return (
                      <Focusable
                        key={`${ach.Name}#${i}`}
                        className={joinClassNames(
                          achievementListClasses.AchievementListItemBase,
                          'sentinel-achievement-item'
                        )}
                        onActivate={() => {}}
                      >
                        {ach.Icon ? (
                          <img
                            src={`${IMG_URL}${ach.Icon}`}
                            alt={ach.DisplayName}
                            className='sentinel-achievement-icon sentinel-achievement-icon--locked'
                          />
                        ) : null}

                        <div className='sentinel-achievement-meta'>
                          <div className={achievementListClasses.AchievementTitle}>{ach.DisplayName}</div>
                          <div
                            className={joinClassNames(
                              achievementListClasses.AchievementDescription,
                              ach.Hidden === 1 ? achievementListClasses.Hidden : ''
                            )}
                          >
                            {ach.Description || ''}
                          </div>
                          {!isLoading && globalPercentages.has(ach.Name) && (
                            <div
                              className={joinClassNames(
                                achievementListClasses.AchievementGlobalPercentage,
                                achievementListClasses.InBody
                              )}
                            >
                              {globalPercentages.get(ach.Name)}% of players have this
                            </div>
                          )}
                        </div>

                        <div className='sentinel-achievement-state'>
                          {hasProgress && (
                            <ProgressBarWithInfo
                              nProgress={Math.round(
                                ((currentAch?.earned && progress !== maxProgress ? progress + 1 : progress) /
                                  maxProgress) *
                                  100
                              )}
                              sOperationText={
                                <div className={achievementListClasses.ProgressCount}>
                                  {currentAch?.earned && progress !== maxProgress ? progress + 1 : progress} /
                                  {maxProgress}
                                </div>
                              }
                              focusable={false}
                            />
                          )}
                        </div>
                      </Focusable>
                    );
                  })}
                </>
              )}
            </div>
          </div>
        </ScrollPanel>
      </div>
    </div>
  );
};

export default AchievementsPage;
