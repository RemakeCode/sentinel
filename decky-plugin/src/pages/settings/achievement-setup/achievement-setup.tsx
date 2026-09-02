import { type FC, useEffect, useRef, useState } from 'react';
import {
  DialogButton,
  DialogButtonPrimary,
  DialogControlsSection,
  DialogControlsSectionHeader,
  Field,
  Focusable,
  showModal,
  Spinner,
  TextField,
  Tabs,
  DialogBody,
  DialogLabel
} from '@decky/ui';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import type { ManagedGBESetupSummary } from '@/shared/types/_generated/sentinel/backend/generator/models';
import type { AppSearchResult } from '@/shared/types/_generated/sentinel/backend/steam/models';
import { GBESetupModal } from '@/pages/settings/achievement-setup/gbe-setup-modal';
import { GBEUndoModal } from '@/pages/settings/achievement-setup/gbe-undo-modal';
import { subscribeGBESetupUpdates } from '@/shared/utils/gbe-setup-events';

const fetcher = new Fetcher();

type SearchStatus = 'idle' | 'loading' | 'results' | 'empty' | 'error';

//language=css
const achievementSetupStyles = `
  .sentinel-achievement-setup-body {
    margin-inline: -2.8vw;
  }

  .sentinel-achievement-search-input {
    position: relative;
  }

  .sentinel-achievement-search-spinner {
    position: absolute;
    inset-inline-end: 14px;
    bottom: 12px;
    width: 18px;
    height: 18px;
    pointer-events: none;
  }

  .sentinel-achievement-selected-game-label {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .sentinel-achievement-selected-game-actions {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .sentinel-achievement-search {
    position: relative;
    box-sizing: border-box;

    & > * {
      box-sizing: inherit;
    }
  }

  .sentinel-achievement-search-results {
    display: flex;
    flex-direction: column;
    gap: 4px;
    width: 100%;
    max-height: 60vh;
    overflow-x: hidden;
    overflow-y: auto;
    position: relative;
    top: -22px;
    box-sizing: inherit;
    background: #262626;
    padding: 0 2px;
    border-bottom-left-radius: 8px;
    border-bottom-right-radius: 8px;
  }

  .sentinel-achievement-search-result {
    display: flex;
    align-items: center;
    gap: 12px;
    box-sizing: inherit;
    height: 68px;
    width: 100%;
    padding: 8px;
    background: hsla(0, 0%, 100%, .1);
  }

  .sentinel-achievement-search-result--focus {
    background: #131313;
    height: inherit;
    width: inherit;
  }

  .sentinel-achievement-search-result-icon {
    flex: 0 0 auto;
    width: 40px;
    height: 40px;
    object-fit: cover;
    border-radius: 2px;
  }

  .sentinel-achievement-search-result-details {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
  }

  .sentinel-achievement-search-result-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
`;

export const AchievementSetupContent: FC = () => {
  const [tab, setTab] = useState<'setup' | 'configured'>('setup');
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<AppSearchResult[]>([]);
  const [selected, setSelected] = useState<AppSearchResult | null>(null);
  const [searchStatus, setSearchStatus] = useState<SearchStatus>('idle');
  const [configured, setConfigured] = useState<ManagedGBESetupSummary[]>([]);
  const searchRequestID = useRef(0);
  const selectDLLButtonRef = useRef<HTMLDivElement>(null);

  const loadConfigured = () => {
    void fetcher
      .get<ManagedGBESetupSummary[]>(`${BASE_URL}/gbe-setup/managed`)
      .then((games) => setConfigured(games ?? []))
      .catch(() => setConfigured([]));
  };

  useEffect(loadConfigured, []);

  useEffect(
    () =>
      subscribeGBESetupUpdates((update) => {
        if (update.phase === 'completed' || update.phase === 'undoCompleted') {
          loadConfigured();
        }
      }),
    []
  );

  useEffect(() => {
    if (!selected) {
      return undefined;
    }

    const animationFrame = requestAnimationFrame(() => selectDLLButtonRef.current?.focus());
    return () => cancelAnimationFrame(animationFrame);
  }, [selected]);

  useEffect(() => {
    const trimmed = query.trim();
    const requestID = ++searchRequestID.current;
    if (trimmed.length < 2) {
      setResults([]);
      setSearchStatus('idle');
      return undefined;
    }

    const timer = setTimeout(() => {
      setSearchStatus('loading');
      void fetcher
        .get<AppSearchResult[]>(`${BASE_URL}/games/search?query=${encodeURIComponent(trimmed)}`)
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
    return () => clearTimeout(timer);
  }, [query]);

  const openSetupModal = () => {
    if (!selected) return;
    let modal: ReturnType<typeof showModal> | undefined;
    let setupReachedTerminalState = false;
    const closeModal = () => {
      modal?.Close();
      loadConfigured();
      if (setupReachedTerminalState) {
        setSelected(null);
      }
    };
    modal = showModal(
      <GBESetupModal
        appId={selected.appId}
        gameName={selected.name}
        closeModal={closeModal}
        onSetupTerminal={() => {
          setupReachedTerminalState = true;
        }}
        subscribeGBESetupUpdates={subscribeGBESetupUpdates}
      />
    );
  };

  const openUndoModal = (game: ManagedGBESetupSummary) => {
    let modal: ReturnType<typeof showModal> | undefined;

    const closeModal = () => {
      modal?.Close();
      loadConfigured();
    };

    modal = showModal(<GBEUndoModal appId={game.appId} gameName={game.name} closeModal={closeModal} />);
  };

  const selectGame = (game: AppSearchResult) => {
    searchRequestID.current++;
    setQuery('');
    setResults([]);
    setSearchStatus('idle');
    setSelected(game);
  };

  const changeGame = () => {
    setSelected(null);
  };

  return (
    <DialogBody className='sentinel-achievement-setup-body'>
      <style>{achievementSetupStyles}</style>
      <Tabs
        activeTab={tab}
        onShowTab={(nextTab: string) => setTab(nextTab as 'setup' | 'configured')}
        tabs={[
          {
            id: 'setup',
            title: 'Setup Achievements',
            content: (
              <DialogControlsSection>
                {!selected && (
                  <>
                    <div className='sentinel-achievement-search'>
                      <div className='sentinel-achievement-search-input'>
                        <TextField
                          value={query}
                          onChange={(event) => {
                            setQuery(event.target.value);
                            setSelected(null);
                          }}
                          label='Search Steam games'
                        />
                        {searchStatus === 'loading' && <Spinner className='sentinel-achievement-search-spinner' />}
                      </div>

                      {searchStatus === 'results' && (
                        <div className='sentinel-achievement-search-results'>
                          {results.map((result) => (
                            <Focusable
                              key={result.appId}
                              className='sentinel-achievement-search-result'
                              focusClassName='sentinel-achievement-search-result--focus'
                              onActivate={() => selectGame(result)}
                            >
                              {result.icon && (
                                <img className='sentinel-achievement-search-result-icon' src={result.icon} alt='' />
                              )}
                              <div className='sentinel-achievement-search-result-details'>
                                <strong className='sentinel-achievement-search-result-name'>{result.name}</strong>
                                <DialogLabel>App ID {result.appId}</DialogLabel>
                              </div>
                            </Focusable>
                          ))}
                        </div>
                      )}
                    </div>
                    {searchStatus === 'empty' && <DialogLabel>No matching games found. Try another title.</DialogLabel>}

                    {searchStatus === 'error' && (
                      <DialogLabel>Steam search is unavailable. Try again later.</DialogLabel>
                    )}
                  </>
                )}
                {selected && (
                  <Field
                    label={
                      <div className='sentinel-achievement-selected-game-label'>
                        <img className='sentinel-achievement-search-result-icon' src={selected.icon} alt='' />
                        <div className='sentinel-achievement-search-result-details'>
                          <div className='sentinel-achievement-search-result-name'>{selected.name}</div>
                          <DialogLabel>{`App ID ${selected.appId}`}</DialogLabel>
                        </div>
                      </div>
                    }
                  >
                    <Focusable
                      className='sentinel-achievement-selected-game-actions'
                      flow-children='down'
                      onCancel={changeGame}
                      onCancelActionDescription='Back'
                    >
                      <DialogButtonPrimary ref={selectDLLButtonRef} onClick={openSetupModal}>
                        Select DLL in Game Folder
                      </DialogButtonPrimary>
                      <DialogButton onClick={changeGame}>Change Game</DialogButton>
                    </Focusable>
                  </Field>
                )}
              </DialogControlsSection>
            )
          },
          {
            id: 'configured',
            title: `Configured Games (${configured.length})`,
            content: (
              <DialogControlsSection>
                <DialogControlsSectionHeader>Configured Games</DialogControlsSectionHeader>
                {configured.length === 0 ? (
                  <Field label='No games have been configured yet.' />
                ) : (
                  configured.map((game) => (
                    <Field key={game.appId} label={game.name}>
                      <DialogButton onClick={() => openUndoModal(game)}>Undo Achievements Setup</DialogButton>
                    </Field>
                  ))
                )}
              </DialogControlsSection>
            )
          }
        ]}
      />
    </DialogBody>
  );
};
