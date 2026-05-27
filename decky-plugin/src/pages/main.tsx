import type { FC } from 'react';
import { useEffect, useState } from 'react';
import {
  ButtonItem,
  focusRingClasses,
  gamepadLibraryClasses,
  libraryAssetImageClasses,
  Navigation,
  PanelSection,
  PanelSectionRow
} from '@decky/ui';
import LibraryImage from '@/shared/components/library-image';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import { getRunningNonSteamGames } from '@/shared/utils/non-steam-game-tracker';
import { getMapping, setMapping } from '@/shared/utils/game-mappings';
import { matchGameByName } from '@/shared/utils/game-matcher';
import type { AchievementInfo, GameBasics } from '@/shared/types/GameBasics';

const fetcher = new Fetcher();

function computeProgress(list: AchievementInfo[]): number {
  if (!list || list.length === 0) return 0;
  const earned = list.filter((a) => a.CurrentAch?.earned).length;
  return Math.round((earned / list.length) * 100);
}

const MainPage: FC = () => {
  const [games, setGames] = useState<GameBasics[]>([]);
  const [matchedGame, setMatchedGame] = useState<GameBasics | null>(null);
  const [loading, setLoading] = useState(true);
  const [screen, setScreen] = useState<'loading' | 'matched' | 'unmatched' | 'empty'>('loading');
  const [runningName, setRunningName] = useState('');

  console.log({ gamepadLibraryClasses, libraryAssetImageClasses, focusRingClasses });

  const matchRunningGame = (gamesList: GameBasics[]) => {
    const running = getRunningNonSteamGames();
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
      setMapping(current.appId, match.AppID);
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
      matchRunningGame(data);
    } catch {
      setScreen('empty');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadGames();
    const interval = setInterval(loadGames, 5000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <PanelSection title='Sentinel'>
        <PanelSectionRow>
          <span>Loading...</span>
        </PanelSectionRow>
      </PanelSection>
    );
  }

  if (screen === 'matched' && matchedGame) {
    const progress = computeProgress(matchedGame.Achievement.List);
    const earned = matchedGame.Achievement.List.filter((a) => a.CurrentAch?.earned).length;

    return (
      <PanelSection title='Now Playing'>
        <PanelSectionRow>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '8px' }}>
            <div style={{ width: '64px', height: '96px', flexShrink: 0 }}>
              <LibraryImage src={matchedGame.PortraitImage} alt={matchedGame.Name} neverShowTitle />
            </div>
            <div style={{ flex: 1 }}>
              <div style={{ fontWeight: 'bold', fontSize: '14px', marginBottom: '4px' }}>{matchedGame.Name}</div>
              <progress value={progress} max={100} style={{ width: '100%', height: '6px' }} />
              <div style={{ fontSize: '12px', color: '#8b929a', marginTop: '4px' }}>
                {progress}% complete — {earned}/{matchedGame.Achievement.List.length} achievements
              </div>
            </div>
          </div>
        </PanelSectionRow>
        <PanelSectionRow>
          <ButtonItem layout='below' onClick={() => Navigation.Navigate(`/sentinel/game/${matchedGame.AppID}`)}>
            View Achievements
          </ButtonItem>
        </PanelSectionRow>
        <PanelSectionRow>
          <ButtonItem layout='below' onClick={() => Navigation.Navigate('/sentinel/library')}>
            Browse Library
          </ButtonItem>
        </PanelSectionRow>
        <PanelSectionRow>
          <ButtonItem layout='below' onClick={() => Navigation.Navigate('/sentinel/settings')}>
            Settings
          </ButtonItem>
        </PanelSectionRow>
      </PanelSection>
    );
  }

  if (screen === 'unmatched') {
    const candidates = matchGameByName(runningName, games) ? games : games.slice(0, 10);

    return (
      <PanelSection title={`Game: ${runningName}`}>
        <PanelSectionRow>
          <span style={{ color: '#8b929a', fontSize: '12px' }}>Pick the matching game from your library</span>
        </PanelSectionRow>
        {candidates.map((game) => {
          const progress = computeProgress(game.Achievement.List);
          return (
            <PanelSectionRow key={game.AppID}>
              <ButtonItem
                layout='below'
                onClick={() => {
                  try {
                    const running = getRunningNonSteamGames()[0];
                    if (running) setMapping(running.appId, game.AppID);
                  } catch {}
                  setMatchedGame(game);
                  setScreen('matched');
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
                  <span>{game.Name}</span>
                  <span style={{ color: '#8b929a', fontSize: '12px' }}>{progress}%</span>
                </div>
              </ButtonItem>
            </PanelSectionRow>
          );
        })}
        <PanelSectionRow>
          <ButtonItem layout='below' onClick={() => Navigation.Navigate('/sentinel/library')}>
            Browse All Games
          </ButtonItem>
        </PanelSectionRow>
      </PanelSection>
    );
  }

  return (
    <PanelSection title='Sentinel'>
      <PanelSectionRow>
        <div style={{ textAlign: 'center', padding: '16px 0', color: '#8b929a' }}>No game running</div>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout='below' onClick={() => Navigation.Navigate('/sentinel/library')}>
          Browse Library
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout='below' onClick={() => Navigation.Navigate('/sentinel/settings')}>
          Settings
        </ButtonItem>
      </PanelSectionRow>
    </PanelSection>
  );
};

export default MainPage;
