import { type Context, type FC, useCallback, useContext, useEffect, useRef, useState } from 'react';
import {
  DialogHeader,
  Focusable,
  Menu,
  MenuItem,
  Navigation,
  ReactRouter,
  ScrollPanel,
  Spinner,
  showContextMenu
} from '@decky/ui';
import { toaster } from '@decky/api';
import { LibraryImage } from '@/shared/components/library-image';
import { EmptyState } from '@/shared/components/empty-state';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import { computeProgress } from '@/shared/utils/utils';
import type { GameBasics } from '@/shared/types/_generated/sentinel/backend/steam/models';
import { decorateGames, type AppConfig, type DeckyGameBasics } from '@/shared/utils/steamgrid';
import { styles } from '@/shared/styles';
import { subscribeLibraryUpdates } from '@/shared/utils/library-update-events';

//language=css
const libraryStyles = `
  .sentinel-library-scroll {
    display: flex;
    flex-direction: column;
    height: calc(100vh - var(--basicui-header-height) - var(--gamepadui-current-footer-height) - 24px);
    min-height: 0;
    margin-inline: -12px;
  }

  .sentinel-library-scroll > div:not(.sentinel-library-header) {
    flex: 1;
    min-height: 0;
    box-sizing: border-box;
    padding: 12px 12px 44px;
    scroll-padding-bottom: 44px;
  }

  .sentinel-library-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, 172px);
    column-gap: 16px;
    row-gap: 42px;
    justify-content: space-between;
  }

  .sentinel-library-header {
    flex-shrink: 0;
    font-size: 24px;
    margin-inline: 12px;
  }

  .sentinel-library-loader {
    width: 100%;
    min-height: 60vh;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .sentinel-library-loader svg {
    width: 48px;
    height: 48px;
  }

  .sentinel-library-sync {
    width: max-content;
    height: 25px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #1c1f24;
    position: fixed;
    transform: translate(-50%, -50%);
    left: 50%;
    padding-inline: 8px;
  }

  .sentinel-library-sync-meta {
    display: flex;
    align-items: center;
    gap: 12px;
    justify-content: space-between;
    font-size: 13px;
    font-weight: 700;
    letter-spacing: 0.03em;
    color: var(--gpColor-Blue, #1a9fff);
  }

  .sentinel-library-sync--error .sentinel-library-sync-meta {
    color: var(--gpColor-Red, #f04747);
  }
`;

const fetcher = new Fetcher();
const SYNC_POLL_INTERVAL_MS = 1000;

interface LibrarySyncStatus {
  State: string;
  Current: number;
  Total: number;
  Failed: number;
}

const emptySyncStatus: LibrarySyncStatus = { State: 'idle', Current: 0, Total: 0, Failed: 0 };

interface LibraryView {
  appId: string;
  scrollTop: number;
}

interface SteamRoute {
  history: {
    action: string;
    location: { pathname: string; state?: { sentinelLibraryView?: LibraryView; [key: string]: unknown } };
    replace(path: string, state: object): void;
  };
}

const SteamRouterContext = Object.values(ReactRouter).find(
  (value) => value !== null && typeof value === 'object' && 'displayName' in value && value.displayName === 'Router'
) as Context<SteamRoute>;

const LibraryPage: FC = () => {
  const { history } = useContext(SteamRouterContext);
  const [returnView] = useState(() =>
    history.action === 'POP' ? history.location.state?.sentinelLibraryView : undefined
  );
  const [restorePending, setRestorePending] = useState(Boolean(returnView));
  const gridRef = useRef<HTMLDivElement>(null);
  const [games, setGames] = useState<DeckyGameBasics[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncStatus, setSyncStatus] = useState<LibrarySyncStatus>(emptySyncStatus);
  const [isFailureDismissed, setIsFailureDismissed] = useState(false);
  const [refreshingGameIds, setRefreshingGameIds] = useState<string[]>([]);
  const lastSyncStatusRef = useRef<LibrarySyncStatus>(emptySyncStatus);

  useEffect(() => {
    if (syncStatus.State !== 'error') {
      setIsFailureDismissed(false);
      return undefined;
    }

    const timeoutId = setTimeout(() => setIsFailureDismissed(true), 10_000);
    return () => clearTimeout(timeoutId);
  }, [syncStatus.State]);

  const loadGames = useCallback(async (showLoading = false, clearOnError = false) => {
    if (showLoading) {
      setLoading(true);
    }

    try {
      const [config, data] = await Promise.all([
        fetcher.get<AppConfig>(`${BASE_URL}/config`),
        fetcher.get<GameBasics[]>(`${BASE_URL}/games`)
      ]);
      const decoratedGames = decorateGames(config, data);
      setGames(decoratedGames);
      return decoratedGames;
    } catch {
      if (clearOnError) {
        setGames([]);
      }
      return [];
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    let active = true;
    let intervalId: ReturnType<typeof setInterval> | undefined;

    const loadSyncStatus = async () => {
      try {
        return await fetcher.get<LibrarySyncStatus>(`${BASE_URL}/games/sync-status`);
      } catch {
        return null;
      }
    };

    const pollSyncStatus = async () => {
      const syncStatus = await loadSyncStatus();
      if (!active || !syncStatus) {
        return;
      }

      const previous = lastSyncStatusRef.current;
      const syncStarted = previous.State !== 'running' && syncStatus.State === 'running';
      const progressed =
        syncStatus.State === 'running' &&
        (syncStarted ? syncStatus.Current > 0 : syncStatus.Current > previous.Current);
      const reachedTerminalState =
        (syncStatus.State === 'done' || syncStatus.State === 'error') &&
        (previous.State !== syncStatus.State ||
          previous.Current !== syncStatus.Current ||
          previous.Total !== syncStatus.Total ||
          previous.Failed !== syncStatus.Failed);

      lastSyncStatusRef.current = syncStatus;
      setSyncStatus(syncStatus);

      if (progressed || reachedTerminalState) {
        await loadGames(false);
      }
    };

    void loadGames(true, true);
    void pollSyncStatus();
    intervalId = setInterval(pollSyncStatus, SYNC_POLL_INTERVAL_MS);

    return () => {
      active = false;
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }, [loadGames]);

  useEffect(() => {
    return subscribeLibraryUpdates(() => {
      void loadGames(false);
    });
  }, [loadGames]);

  const handleRefreshGame = async (appId: string) => {
    if (!appId || refreshingGameIds.includes(appId)) {
      return;
    }

    setRefreshingGameIds((current) => [...current, appId]);

    try {
      const [config, refreshedGame] = await Promise.all([
        fetcher.get<AppConfig>(`${BASE_URL}/config`),
        fetcher.post<GameBasics>(`${BASE_URL}/games/${appId}/refresh`, {})
      ]);
      const decoratedGame = decorateGames(config, [refreshedGame])[0];
      setGames((current) =>
        current.map((game) => {
          if (game.AppID !== decoratedGame.AppID) {
            return game;
          }

          return decoratedGame;
        })
      );
      toaster.toast({ title: 'Success', body: `${refreshedGame.Name || 'Game'} refreshed` });
    } catch {
      toaster.toast({ title: 'Error', body: 'Failed to refresh game' });
    } finally {
      setRefreshingGameIds((current) => current.filter((id) => id !== appId));
    }
  };

  const openGameContextMenu = (appId: string, parent?: EventTarget | null) => {
    showContextMenu(
      <Menu label='Game Actions'>
        <MenuItem
          disabled={refreshingGameIds.includes(appId)}
          onClick={() => {
            void handleRefreshGame(appId);
          }}
        >
          {refreshingGameIds.includes(appId) ? 'Refreshing...' : 'Refresh Game'}
        </MenuItem>
      </Menu>,
      parent ?? undefined
    );
  };

  const isSyncRunning = syncStatus.State === 'running';
  const isSyncError = syncStatus.State === 'error';
  const syncMessage = isSyncError
    ? syncStatus.Failed > 0
      ? 'Metadata sync completed with failures'
      : 'Metadata sync failed'
    : 'Fetching metadata';
  const syncCount =
    isSyncError && syncStatus.Failed > 0
      ? `${syncStatus.Failed}/${syncStatus.Total} failed`
      : `${syncStatus.Current}/${syncStatus.Total}`;
  const showInitialSpinner = loading || (isSyncRunning && games.length < 1);
  const showEmptyState = !showInitialSpinner && games.length === 0;
  const returnGameId = games.some((game) => game.AppID === returnView?.appId) ? returnView?.appId : undefined;

  useEffect(() => {
    if (!restorePending || showInitialSpinner) return;
    const scroller = gridRef.current?.parentElement;
    if (!scroller) return;
    scroller.scrollTop = returnGameId ? (returnView?.scrollTop ?? 0) : 0;
    if (!returnGameId) setRestorePending(false);
  }, [restorePending, showInitialSpinner, returnGameId, returnView]);

  const openGame = (appId: string) => {
    history.replace(history.location.pathname, {
      ...history.location.state,
      sentinelLibraryView: { appId, scrollTop: gridRef.current?.parentElement?.scrollTop ?? 0 }
    });
    Navigation.Navigate(`/sentinel/games/${appId}`);
  };

  return (
    <div style={styles.wrapper}>
      <style>{libraryStyles}</style>
      {(isSyncRunning || (isSyncError && !isFailureDismissed)) && (
        <div
          className={`sentinel-library-sync${isSyncError ? ' sentinel-library-sync--error' : ''}`}
          aria-live='polite'
          aria-busy={isSyncRunning}
        >
          <div className='sentinel-library-sync-meta'>
            <span>{syncMessage}</span>
            <span>{syncCount}</span>
          </div>
        </div>
      )}
      <div className='sentinel-library-scroll'>
        {!showInitialSpinner && !showEmptyState && (
          <DialogHeader className={'sentinel-library-header'}>Library</DialogHeader>
        )}
        <ScrollPanel {...{ fnScrollIntoViewHandler: restorePending && returnGameId ? () => true : undefined }}>
          {showInitialSpinner ? (
            <div className='sentinel-library-loader' aria-busy='true'>
              <Spinner />
            </div>
          ) : showEmptyState ? (
            <EmptyState
              variant='library'
              label={isSyncError ? 'Game metadata could not be loaded' : 'No games found'}
              buttonText='Go to Settings'
              buttonClick={() => Navigation.Navigate('/sentinel/settings')}
            />
          ) : (
            <>
              <Focusable className='sentinel-library-grid' ref={gridRef} autoFocus={!returnGameId}>
                {games.map((game) => {
                  const progress = computeProgress(game.Achievement.List);
                  const isRefreshing = refreshingGameIds.includes(game.AppID);
                  return (
                    <LibraryImage
                      key={game.AppID}
                      steamCard
                      autoFocus={game.AppID === returnGameId}
                      onGamepadFocus={
                        restorePending && game.AppID === returnGameId
                          ? () => {
                              const scroller = gridRef.current?.parentElement;
                              if (scroller) scroller.scrollTop = returnView?.scrollTop ?? 0;
                              setRestorePending(false);
                            }
                          : undefined
                      }
                      src={game.PortraitImage}
                      fallbackSrc={game.FallbackPortraitImage}
                      alt={game.Name}
                      name={game.Name}
                      progress={progress}
                      isRefreshing={isRefreshing}
                      onActivate={() => openGame(game.AppID)}
                      onOpenContextMenu={(parent) => void openGameContextMenu(game.AppID, parent)}
                    />
                  );
                })}
              </Focusable>
            </>
          )}
        </ScrollPanel>
      </div>
    </div>
  );
};

export default LibraryPage;
