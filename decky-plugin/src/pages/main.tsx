import type { FC } from 'react';
import { useEffect, useState } from 'react';
import {
  achievementListClasses,
  ButtonItem,
  Focusable,
  joinClassNames,
  Marquee,
  Navigation,
  PanelSection,
  PanelSectionRow,
  ProgressBar
} from '@decky/ui';
import { FaUnlock } from 'react-icons/fa';
import { LibraryImage } from '@/shared/components/library-image';
import { EmptyState } from '@/shared/components/empty-state';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import { runningGames, subscribeToGameChanges } from '@/shared/utils/non-steam-game-tracker';
import { getMapping, setMapping } from '@/shared/utils/game-mappings';
import { matchGameByName } from '@/shared/utils/game-matcher';
import { showConfirmModal } from '@/shared/components/confirm';
import { computeProgress } from '@/shared/utils/utils';
import type { GameBasics } from '@/shared/types/GameBasics';
import { ImgIcon } from '@/shared/components/img-icon';

const fetcher = new Fetcher();

//language=css
const mainStyles = `
  .sentinel-qam-scroll-area {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .sentinel-qam-header-card {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding-bottom: 6px;
    margin-bottom: 6px;
    border-bottom: 1px solid hsla(0, 0%, 100%, .1);
  }

  .sentinel-qam-header {
    text-transform: uppercase;
    font-size: 16px;
    font-weight: bold;
    opacity: 0.7;
  }

  .sentinel-qam-game-content {
    display: flex;
    flex-direction: column;
    flex: 1;
  }

  .sentinel-qam-game-title {
    font-size: 12px;
    line-height: 1.25;
    text-transform: uppercase;
    font-weight: 700;
    margin-block-end: 8px;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sentinel-qam-game-image {
    width: 50px;
    height: 75px;
  }

  .sentinel-qam-progress-count {
    align-self: flex-end;
  }

  .sentinel-qam-ach-item {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    height: 50px;
    padding: 5px;
    border: 1px solid hsla(0, 0%, 100%, .1);
    background: hsla(0, 0%, 100%, .05);
    border-radius: 4px;
  }

  .sentinel-qam-ach-item--focus, .sentinel-qam-header-card--focus {
    background: hsla(0, 0%, 10%, 0.15);
  }

  .sentinel-qam-ach-image {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 50px;
    height: 50px;
    & > img {
      position:relative;
      bottom:0;
      height:48px;
    }
  }

  .sentinel-qam-ach-content {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
  }

  .sentinel-qam-ach-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .sentinel-qam-ach-name {
    font-weight: bold;
    font-size: 13px;
    text-overflow: ellipsis;
    overflow: hidden;
    white-space: nowrap;
  }

  .sentinel-qam-ach-icon-unlocked {
    fill: #4ade80;
    flex-shrink: 0;
  }

  .sentinel-qam-ach-description {
    font-size: 12px;
    color: #8b929a;
    text-overflow: ellipsis;
    overflow: hidden;
    white-space: nowrap;
    display: block;
  }
  .sentinel-qam-ach-description-hidden {
    filter: blur(4px);
    cursor: pointer;
    transition: filter 200ms linear;

    > :hover {
      filter: none
    }
  }

  .sentinel-qam-ach-progress {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    position: relative;
    margin-block-start: 6px;
  }

  .sentinel-qam-ach-progress-text {
    font-size: 11px;
    color: #8b929a;
    text-align: right;
    position: absolute;
    right: 0;
    top: -15px;
    z-index: 1;
  }
`;

const MainPage: FC = () => {
  const [games, setGames] = useState<GameBasics[]>([]);
  const [matchedGame, setMatchedGame] = useState<GameBasics | null>(null);
  const [loading, setLoading] = useState(true);
  const [screen, setScreen] = useState<'loading' | 'matched' | 'unmatched' | 'empty'>('loading');
  const [runningName, setRunningName] = useState('');
  const [playingKey, setPlayingKey] = useState<string | null>(null);

  const playMarquee = (key: string, play: boolean) => {
    setPlayingKey((prev) => (play ? key : prev === key ? null : prev));
  };

  const selectGame = async (game: GameBasics) => {
    const confirmed = await showConfirmModal({
      title: 'Confirm Mapping',
      description: `Are you sure the game title is ${game.Name}?`,
      okText: "Yes, I'm sure",
      cancelText: 'Cancel'
    });
    if (!confirmed) return;
    try {
      const running = runningGames()[0];
      if (running) setMapping(running.appId, game.AppID, game.Name, running.name);
    } catch {}
    setMatchedGame(game);
    setScreen('matched');
  };

  const matchRunningGame = (gamesList: GameBasics[]) => {
    const running = runningGames();
    const current = running[0] ?? null;

    if (!current) {
      setScreen('empty');
      return;
    }

    setRunningName(current.name);

    const mappedId = getMapping(current.appId);
    if (mappedId) {
      const found = gamesList.find((g) => g.AppID === mappedId);
      if (found) {
        setMatchedGame(found);
        setScreen('matched');
        return;
      }
    }

    const match = matchGameByName(current.name, gamesList);

    if (match) {
      setMapping(current.appId, match.AppID, match.Name, current.name);
      setMatchedGame(match);
      setScreen('matched');
      return;
    }

    setScreen('unmatched');
  };

  const loadGames = async () => {
    try {
      const data = await fetcher.get<GameBasics[]>(`${BASE_URL}/games`);
      setGames(data);
    } catch {
      setScreen('empty');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadGames();
  }, []);

  useEffect(() => {
    matchRunningGame(games);
    const unsubscribe = subscribeToGameChanges(() => {
      matchRunningGame(games);
    });
    return unsubscribe;
  }, [games]);

  // DEV: seed a fake running game for testing without Steam
  // useEffect(() => {
  //   processAppOverviewChange({
  //     app_overview: [
  //       {
  //         appid: 3009130864,
  //         display_name: 'Shadow of Mordor',
  //         app_type: 1073741824,
  //         per_client_data: [{ display_status: 4, is_available_on_current_platform: true }]
  //       }
  //     ],
  //     removed_appid: []
  //   });
  // }, []);

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', padding: '16px 0' }}>
        Loading
        {/*TODO- CHange this*/}
      </div>
    );
  }

  if (screen === 'matched' && matchedGame) {
    const progress = computeProgress(matchedGame.Achievement.List);
    const earned = matchedGame.Achievement.List.filter((a) => a.CurrentAch?.earned).length;
    const achievements = matchedGame.Achievement.List;

    return (
      <PanelSection>
        <style>{mainStyles}</style>
        <div className='sentinel-qam-scroll-area'>
          <Focusable
            className='sentinel-qam-header-card'
            onActivate={() => {}}
            focusClassName='sentinel-qam-header-card--focus'
          >
            <div className='sentinel-qam-header'>Now Playing</div>
            <div style={{ display: 'flex', gap: '8px' }}>
              <div className='sentinel-qam-game-image'>
                <LibraryImage src={matchedGame.PortraitImage} alt={matchedGame.Name} />
              </div>
              <div className='sentinel-qam-game-content'>
                <div className='sentinel-qam-game-title'>{matchedGame.Name}</div>
                <ProgressBar nProgress={progress} focusable={false} />
                <div className={joinClassNames(achievementListClasses.ProgressCount, 'sentinel-qam-progress-count')}>
                  <strong>{progress}% complete</strong> - {earned}/{matchedGame.Achievement.List.length}
                </div>
              </div>
            </div>
          </Focusable>

          {achievements.map((ach, i) => {
            const key = `${ach.Name}#${i}`;
            const earnedAch = ach.CurrentAch?.earned;
            const hasProgress = (ach.CurrentAch?.max_progress || 0) > 1;
            const currentProgress = ach.CurrentAch?.progress || 0;
            const maxProgress = ach.CurrentAch?.max_progress || 1;
            const isPlaying = playingKey === key;

            return (
              <Focusable
                key={key}
                onActivate={() => {}}
                onFocus={() => playMarquee(key, true)}
                onBlur={() => playMarquee(key, false)}
                focusClassName='sentinel-qam-ach-item--focus'
                className={joinClassNames('sentinel-qam-ach-item')}
              >
                <div className={'sentinel-qam-ach-image'}>
                  <ImgIcon src={ach.Icon} style={{ bottom: 0, height: '48px' }} />
                </div>
                <div className='sentinel-qam-ach-content'>
                  <div className='sentinel-qam-ach-row'>
                    <div className='sentinel-qam-ach-name'>{ach.DisplayName}</div>
                    {earnedAch && <FaUnlock className='sentinel-qam-ach-icon-unlocked' size={12} />}
                  </div>
                  {!earnedAch && hasProgress ? (
                    <div className='sentinel-qam-ach-progress'>
                      <span className='sentinel-qam-ach-progress-text'>
                        {currentProgress}/{maxProgress}
                      </span>
                      <ProgressBar nProgress={Math.round((currentProgress / maxProgress) * 100)} focusable={false} />
                    </div>
                  ) : ach.Hidden === 1 ? (
                    <div
                      className={joinClassNames('sentinel-qam-ach-description', 'sentinel-qam-ach-description-hidden')}
                    >
                      {ach.Description || ''}
                    </div>
                  ) : (
                    <Marquee className='sentinel-qam-ach-description' play={isPlaying} delay={1} resetOnPause={true}>
                      {ach.Description || ''}
                    </Marquee>
                  )}
                </div>
              </Focusable>
            );
          })}
        </div>
      </PanelSection>
    );
  }

  if (screen === 'unmatched') {
    const candidates = matchGameByName(runningName, games) ? games : games.slice(0, 10);

    return (
      <PanelSection title={'Pick matching game title'}>
        {candidates.map((game) => {
          const gameKey = `game-${game.AppID}`;
          return (
            <PanelSectionRow key={game.AppID}>
              <div onFocus={() => playMarquee(gameKey, true)} onBlur={() => playMarquee(gameKey, false)}>
                <ButtonItem layout='below' onClick={() => selectGame(game)}>
                  <Marquee play={playingKey === gameKey} resetOnPause={true} delay={1}>
                    {game.Name}
                  </Marquee>
                </ButtonItem>
              </div>
            </PanelSectionRow>
          );
        })}
        <PanelSectionRow>
          <ButtonItem layout={'below'} onClick={() => Navigation.Navigate('/sentinel/library')}>
            Browse All Games
          </ButtonItem>
        </PanelSectionRow>
      </PanelSection>
    );
  }

  return (
    <PanelSection>
      <EmptyState
        variant='main'
        label='No game running'
        buttonText='Browse Library'
        buttonClick={() => Navigation.Navigate('/sentinel/library')}
      />
    </PanelSection>
  );
};

export default MainPage;
