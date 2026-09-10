import './achievement-setup.scss';
import { useEffect, useRef, useState, type FC } from 'react';
import { Trophy } from 'lucide-react';
import { Events } from '@wailsio/runtime';
import { SearchApps } from '@wa/sentinel/backend/steam/service';
import type { AppSearchResult } from '@wa/sentinel/backend/steam/models';
import { ManagedGBESetups } from '@wa/sentinel/backend/generator/service';
import type { ManagedGBESetupSummary } from '@wa/sentinel/backend/generator/models';
import EmptyState from '@/shared/components/empty-state';
import { GBESetupModal } from './gbe-setup-modal';
import { GBEUndoModal } from './gbe-undo-modal';

type SearchStatus = 'idle' | 'loading' | 'results' | 'empty' | 'error';

const AchievementSetup: FC = () => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<AppSearchResult[]>([]);
  const [searchStatus, setSearchStatus] = useState<SearchStatus>('idle');
  const [selected, setSelected] = useState<AppSearchResult | null>(null);
  const [configured, setConfigured] = useState<ManagedGBESetupSummary[]>([]);
  const [setupGame, setSetupGame] = useState<AppSearchResult | null>(null);
  const [setupReachedTerminalState, setSetupReachedTerminalState] = useState(false);
  const [undoGame, setUndoGame] = useState<ManagedGBESetupSummary | null>(null);
  const searchRequestID = useRef(0);
  const selectDLLButtonRef = useRef<HTMLButtonElement>(null);

  const loadConfigured = () => {
    void ManagedGBESetups()
      .then((games) => setConfigured(games ?? []))
      .catch(() => setConfigured([]));
  };

  useEffect(loadConfigured, []);

  useEffect(() => {
    return Events.On('sentinel::achievement-setup-update', (event: { data: { phase: string } }) => {
      if (event.data.phase === 'completed' || event.data.phase === 'undoCompleted') loadConfigured();
    });
  }, []);

  useEffect(() => {
    const trimmed = query.trim();
    const requestID = ++searchRequestID.current;
    if (trimmed.length < 2) {
      setResults([]);
      setSearchStatus('idle');
      return;
    }
    const timer = window.setTimeout(() => {
      if (requestID !== searchRequestID.current) return;
      setSearchStatus('loading');
      void SearchApps(trimmed)
        .then((games) => {
          if (requestID !== searchRequestID.current) return;
          const nextResults = games ?? [];
          setResults(nextResults);
          setSearchStatus(nextResults.length > 0 ? 'results' : 'empty');
        })
        .catch(() => {
          if (requestID !== searchRequestID.current) return;
          setResults([]);
          setSearchStatus('error');
        });
    }, 300);
    return () => window.clearTimeout(timer);
  }, [query]);

  useEffect(() => {
    if (!selected) return undefined;

    const animationFrame = window.requestAnimationFrame(() => selectDLLButtonRef.current?.focus());
    return () => window.cancelAnimationFrame(animationFrame);
  }, [selected]);

  const clearSearch = () => {
    searchRequestID.current++;
    setQuery('');
    setResults([]);
    setSearchStatus('idle');
  };

  const selectGame = (game: AppSearchResult) => {
    clearSearch();
    setSelected(game);
  };

  const changeGame = () => {
    setSelected(null);
    clearSearch();
  };

  const changeQuery = (nextQuery: string) => {
    searchRequestID.current++;
    setQuery(nextQuery);
    setSelected(null);
    setResults([]);
    setSearchStatus('idle');
  };

  const openSetupModal = () => {
    if (!selected) return;
    setSetupReachedTerminalState(false);
    setSetupGame(selected);
  };

  const closeSetupModal = () => {
    setSetupGame(null);
    loadConfigured();
    if (setupReachedTerminalState) {
      setSetupReachedTerminalState(false);
      changeGame();
    }
  };

  return (
    <section className='settings-pane page-content'>
      <div className='card settings-section settings-section-first achievement-setup-content'>
        <h4 className='settings-section-title'>
          <Trophy /> <span>Achievement Setup</span>
        </h4>
        <hr className='divider' />
        <ot-tabs data-anchor='achievement-setup'>
          <div role='tablist'>
            <button role='tab' id='setup' aria-selected='true'>
              Setup Achievements
            </button>
            <button role='tab' id='configured-games'>
              Configured Games ({configured.length})
            </button>
          </div>
          <section role='tabpanel'>
            {!selected && (
              <>
                <div className='achievement-setup-search'>
                  <label data-field>
                    Search game name on Steam
                    <div className='achievement-setup-search-control'>
                      <input
                        autoComplete='off'
                        autoCapitalize='off'
                        autoCorrect='off'
                        autoFocus
                        value={query}
                        onChange={(event) => changeQuery(event.target.value)}
                        placeholder='Search Steam games'
                      />
                      {searchStatus === 'loading' && (
                        <span
                          className='achievement-setup-search-loading'
                          role='status'
                          aria-busy='true'
                          data-spinner='small'
                          aria-label='Searching'
                        />
                      )}
                    </div>
                    {searchStatus === 'empty' && (
                      <div className='achievement-setup-search-error'>No matching games found. Try another title.</div>
                    )}
                    {searchStatus === 'error' && (
                      <div className='achievement-setup-search-error'>
                        Steam search is unavailable. Try again later.
                      </div>
                    )}
                  </label>
                  {searchStatus === 'results' && (
                    <div className='achievement-setup-results' role='listbox' aria-label='Steam search results'>
                      {results.map((result) => (
                        <button
                          type='button'
                          role='option'
                          className='outline'
                          key={result.appId}
                          onClick={() => selectGame(result)}
                        >
                          <img src={result.icon} alt='' />
                          <div className='flex flex-col'>
                            <span>{result.name}</span>
                            <small>App ID {result.appId}</small>
                          </div>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              </>
            )}
            {selected && (
              <div className='achievement-setup-selected'>
                <img src={selected.icon} alt='' />

                <div className='flex flex-col'>
                  <span className='achievement-setup-selected-title'>{selected.name}</span>
                  <small className='text-light'>App ID {selected.appId}</small>
                </div>
                <div className='achievement-setup-selected-actions'>
                  <button ref={selectDLLButtonRef} type='button' onClick={openSetupModal}>
                    Select DLL in Game Folder
                  </button>
                  <button type='button' className='outline' onClick={changeGame}>
                    Change Game
                  </button>
                </div>
              </div>
            )}
          </section>
          <section role='tabpanel' aria-label='Configured games'>
            <div className='settings-grid'>
              {configured.length === 0 && <EmptyState message='No games have been configured yet.' />}

              {configured.length > 0 &&
                configured.map((game) => (
                  <div className='settings-grid-item achievement-setup-configured-row' key={game.appId}>
                    <span className='badge' data-variant='success'>Configured</span>
                    <div className='achievement-setup-configured-details'>
                      <strong>{game.name}</strong>
                      <small>App ID {game.appId}</small>
                    </div>
                    <div className='settings-grid-actions'>
                      <button type='button' className='outline' onClick={() => setUndoGame(game)}>
                        Undo Achievements Setup
                      </button>
                    </div>
                    <div />
                  </div>
                ))}
            </div>
          </section>
        </ot-tabs>
        {setupGame && (
          <GBESetupModal
            isOpen
            appId={setupGame.appId}
            gameName={setupGame.name}
            onSetupTerminal={() => setSetupReachedTerminalState(true)}
            onClose={closeSetupModal}
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
    </section>
  );
};

export default AchievementSetup;
