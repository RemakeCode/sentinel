import './achievement-setup.scss';
import { useEffect, useState, type FC } from 'react';
import { Search } from 'lucide-react';
import { Events } from '@wailsio/runtime';
import { SearchApps } from '@wa/sentinel/backend/steam/service';
import type { AppSearchResult } from '@wa/sentinel/backend/steam/models';
import { ManagedGBESetups } from '@wa/sentinel/backend/generator/service';
import type { ManagedGBESetupSummary } from '@wa/sentinel/backend/generator/models';
import { GBESetupModal } from '@/shared/components/achievement-setup/gbe-setup-modal';
import { GBEUndoModal } from '@/shared/components/achievement-setup/gbe-undo-modal';

type Tab = 'setup' | 'configured';

export const AchievementSetupContent: FC = () => {
  const [tab, setTab] = useState<Tab>('setup');
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<AppSearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState('');
  const [selected, setSelected] = useState<AppSearchResult | null>(null);
  const [configured, setConfigured] = useState<ManagedGBESetupSummary[]>([]);
  const [setupGame, setSetupGame] = useState<AppSearchResult | null>(null);
  const [undoGame, setUndoGame] = useState<ManagedGBESetupSummary | null>(null);

  const loadConfigured = () => {
    void ManagedGBESetups()
      .then((games) => setConfigured(games ?? []))
      .catch(() => setConfigured([]));
  };

  useEffect(loadConfigured, []);

  useEffect(() => {
    return Events.On('sentinel::achievement-setup-update', (event: { data: { phase: string } }) => {
      if (event.data.phase === 'completed' || event.data.phase === 'undoCompleted') {
        loadConfigured();
      }
    });
  }, []);

  useEffect(() => {
    const trimmed = query.trim();
    if (trimmed.length < 2) {
      setResults([]);
      setSearching(false);
      setSearchError('');
      return;
    }
    const timer = window.setTimeout(() => {
      setSearching(true);
      setSearchError('');
      void SearchApps(trimmed)
        .then((games) => setResults(games ?? []))
        .catch(() => {
          setResults([]);
          setSearchError('Unable to search Steam right now. Try again later.');
        })
        .finally(() => setSearching(false));
    }, 300);
    return () => window.clearTimeout(timer);
  }, [query]);

  return (
    <div className='achievement-setup-content'>
      <ot-tabs className='achievement-setup-tabs'>
        <div role='tablist' aria-label='Achievement setup sections'>
          <button
            role='tab'
            aria-selected={tab === 'setup'}
            className={tab === 'setup' ? 'active' : ''}
            onClick={() => setTab('setup')}
          >
            Setup
          </button>
          <button
            role='tab'
            aria-selected={tab === 'configured'}
            className={tab === 'configured' ? 'active' : ''}
            onClick={() => setTab('configured')}
          >
            Configured Games ({configured.length})
          </button>
        </div>

        <section role='tabpanel' aria-label='Setup' hidden={tab !== 'setup'}>
          <h2>Setup Achievements For a Game</h2>
          <p className='achievement-setup-help'>
            Search Steam for a game, choose its installation DLL, and continue through setup.
          </p>
          <div className='achievement-setup-search'>
            <Search size={18} aria-hidden='true' />
            <input
              value={query}
              onChange={(event) => {
                setQuery(event.target.value);
                setSelected(null);
              }}
              placeholder='Search Steam games'
              aria-label='Search Steam games'
            />
          </div>
          {searching && <p className='achievement-setup-muted'>Searching Steam…</p>}
          {searchError && <p className='achievement-setup-error'>{searchError}</p>}
          {results.length > 0 && (
            <div className='achievement-setup-results' role='listbox' aria-label='Steam search results'>
              {results.map((result) => (
                <button
                  type='button'
                  role='option'
                  aria-selected={selected?.appId === result.appId}
                  className={selected?.appId === result.appId ? 'selected' : ''}
                  key={result.appId}
                  onClick={() => setSelected(result)}
                >
                  {result.icon ? (
                    <img src={result.icon} alt='' />
                  ) : (
                    <span className='achievement-setup-icon-placeholder' />
                  )}
                  <span>{result.name}</span>
                  <small>App ID {result.appId}</small>
                </button>
              ))}
            </div>
          )}
          {selected && (
            <div className='achievement-setup-selected'>
              <div>
                <strong>{selected.name}</strong>
                <span>App ID {selected.appId}</span>
              </div>
              <button type='button' onClick={() => setSetupGame(selected)}>
                Select DLL and continue
              </button>
            </div>
          )}
        </section>

        <section role='tabpanel' aria-label='Configured games' hidden={tab !== 'configured'}>
          <h2>Configured Games</h2>
          {configured.length === 0 ? (
            <p className='achievement-setup-muted'>No games have been configured yet.</p>
          ) : (
            <div className='achievement-setup-configured'>
              {configured.map((game) => (
                <div className='achievement-setup-configured-row' key={game.appId}>
                  <span>{game.name}</span>
                  <button type='button' onClick={() => setUndoGame(game)}>
                    Undo Achievements Setup
                  </button>
                </div>
              ))}
            </div>
          )}
        </section>
      </ot-tabs>
      {setupGame && (
        <GBESetupModal
          isOpen
          appId={setupGame.appId}
          gameName={setupGame.name}
          onClose={() => {
            setSetupGame(null);
            loadConfigured();
          }}
        />
      )}
      {undoGame && (
        <GBEUndoModal
          isOpen
          appId={undoGame.appId}
          gameName={undoGame.name}
          onClose={() => {
            setUndoGame(null);
            loadConfigured();
          }}
        />
      )}
    </div>
  );
};
