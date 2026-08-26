import { type FC, useEffect, useState } from 'react';
import { FileSelectionType, openFilePicker } from '@decky/api';
import { ConfirmModal, DialogBody, DialogBodyText, ProgressBar, showModal } from '@decky/ui';
import { renderSVG } from 'uqr';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';

export interface GBESetupUpdate {
  phase:
    | 'preparing'
    | 'awaitingQr'
    | 'generating'
    | 'installing'
    | 'completed'
    | 'cancelled'
    | 'timedOut'
    | 'failed'
    | 'undoCompleted';
  message?: string;
  challengeUrl?: string;
  gseVersion?: string;
  gbeVersion?: string;
}

interface BackupStatus {
  hasDllBackup: boolean;
  hasSettingsBackup: boolean;
}

interface SetupProgressStage {
  value: number;
  step: number;
  label: string;
}

const dllSelectionStage: SetupProgressStage = { value: 15, step: 1, label: 'Select Steam API DLL' };

type SetupState =
  | { kind: 'confirm'; progressStage: SetupProgressStage }
  | {
      kind: 'running' | 'cancelling';
      update?: GBESetupUpdate;
      progressStage: SetupProgressStage;
      challengeURL?: string;
    }
  | { kind: 'terminal'; update: GBESetupUpdate; progressStage: SetupProgressStage; challengeURL?: string };
type UndoState = { kind: 'confirm' } | { kind: 'restoring' } | { kind: 'terminal'; update: GBESetupUpdate };
type GBESetupHandler = (update: GBESetupUpdate) => void;

const fetcher = new Fetcher();

let activeGBESetupHandler: GBESetupHandler | null = null;

function progressStageFor(phase: GBESetupUpdate['phase']): SetupProgressStage | undefined {
  switch (phase) {
    case 'preparing':
      return { value: 15, step: 1, label: 'Preparing tools' };
    case 'awaitingQr':
      return { value: 35, step: 2, label: 'Waiting for Steam approval' };
    case 'generating':
      return { value: 65, step: 3, label: 'Generating configuration' };
    case 'installing':
      return { value: 85, step: 4, label: 'Installing setup' };
    case 'completed':
      return { value: 100, step: 4, label: 'Setup complete' };
    default:
      return undefined;
  }
}

function progressLabelFor(state: SetupState): string {
  switch (state.kind) {
    case 'cancelling':
      return 'Cancelling setup';

    case 'terminal':
      switch (state.update.phase) {
        case 'completed':
          return 'Setup complete';
        case 'cancelled':
          return 'Setup cancelled';
        case 'timedOut':
          return 'Sign-in timed out';
        case 'failed':
          return 'Setup failed';
      }
  }

  return `${state.progressStage.step} of 4 · ${state.progressStage.label}`;
}

function showsSetupTitleProgress(state: SetupState): boolean {
  return state.kind !== 'terminal' || (state.update.phase !== 'cancelled' && state.update.phase !== 'timedOut');
}

function setupPrimaryActionLabel(state: SetupState): string {
  switch (state.kind) {
    case 'confirm':
      return 'Continue';
    case 'running':
      return 'Cancel';
    case 'cancelling':
      return 'Cancelling…';
    case 'terminal':
      return 'Close';
  }
}

function applySetupUpdate(state: SetupState, update: GBESetupUpdate): SetupState {
  if (update.phase === 'undoCompleted') {
    return state;
  }

  const progressStage = progressStageFor(update.phase) ?? state.progressStage;
  const challengeURL = update.challengeUrl ?? ('challengeURL' in state ? state.challengeURL : undefined);

  if (
    update.phase === 'completed' ||
    update.phase === 'failed' ||
    update.phase === 'cancelled' ||
    update.phase === 'timedOut'
  ) {
    return { kind: 'terminal', update, progressStage, challengeURL };
  }

  return {
    kind: state.kind === 'cancelling' ? 'cancelling' : 'running',
    update,
    progressStage,
    challengeURL
  };
}

export const dispatchGBESetupUpdate = (update: GBESetupUpdate) => {
  activeGBESetupHandler?.(update);
};

const subscribeGBESetupUpdates = (handler: GBESetupHandler) => {
  activeGBESetupHandler = handler;
  return () => {
    if (activeGBESetupHandler === handler) {
      activeGBESetupHandler = null;
    }
  };
};

//language=css
const setupModalStyles = `
  .sentinel-gbe-setup-content {
    box-sizing: border-box;
    height: 160px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .sentinel-gbe-setup-qr {
    display: grid;
    grid-template-columns: 160px minmax(0, 1fr);
    align-items: center;
    gap: 16px;
    min-height: 160px;
  }

  .sentinel-gbe-setup-qr-code {
    overflow: hidden;
    background: white;
    border-radius: 6px;
  }

  .sentinel-gbe-setup-qr-image,
  .sentinel-gbe-setup-qr-image svg {
    width: 100%;
    height: 100%;
    display: block;
  }

  .sentinel-gbe-setup-qr-image--blurred {
    filter: blur(6px);
    transform: scale(1.04);
  }

  .sentinel-gbe-setup-qr-message {
    overflow-wrap: anywhere;
    text-align: center;
  }

  .sentinel-gbe-setup-qr-details {
    align-self: stretch;
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 12px;
  }

  .sentinel-gbe-setup-title {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    width: 100%;
    padding-bottom: 8px;
    border-bottom: 1px solid rgba(100, 100, 100, 0.4);
  }

  .sentinel-gbe-setup-title-text {
    flex: 1;
    min-width: 0;
  }

  .sentinel-gbe-setup-progress {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 4px;
    width: min(46%, 240px);
    flex-shrink: 0;
    text-align: right;
    color: var(--gpColor-Blue, #1a9fff);
    font-size: 12px;
    line-height: 12px;
    text-transform: uppercase;
  }

  .sentinel-gbe-setup-progress > :last-child {
    width: 100%;
  }

  .sentinel-gbe-setup-progress progress {
    width: 100%;
    accent-color: var(--gpColor-Blue, #1a9fff);
    font-weight: 700;
  }

  @media (max-width: 600px) {
    .sentinel-gbe-setup-title {
      flex-direction: column;
    }

    .sentinel-gbe-setup-progress {
      width: 100%;
      align-items: stretch;
      text-align: left;
    }
  }

  .sentinel-gbe-undo-content {
    box-sizing: border-box;
    min-height: 0;
    display: flex;
    align-items: flex-start;
  }

  .sentinel-gbe-undo-modal {
    width: min(420px, calc(100vw - 32px));
  }
`;

const QRCode: FC<{ value: string; size: number; blurred?: boolean }> = ({
  value,
  size: renderedSize,
  blurred = false
}) => {
  const svg = renderSVG(value, { border: 2 });

  return (
    <div
      className='sentinel-gbe-setup-qr-code'
      role='img'
      aria-label='Steam sign-in QR code'
      style={{ width: renderedSize, height: renderedSize }}
    >
      <div
        className={`sentinel-gbe-setup-qr-image${blurred ? ' sentinel-gbe-setup-qr-image--blurred' : ''}`}
        dangerouslySetInnerHTML={{ __html: svg }}
      />
    </div>
  );
};

const SetupTitle: FC<{ stage?: SetupProgressStage; label: string; showProgress?: boolean }> = ({
  stage,
  label,
  showProgress = true
}) => (
  <div className='sentinel-gbe-setup-title'>
    <span className='sentinel-gbe-setup-title-text'>Setup Achievements</span>
    {showProgress && (
      <div className='sentinel-gbe-setup-progress'>
        <span>{label}</span>
        <ProgressBar nProgress={stage?.value ?? 0} focusable={false} />
      </div>
    )}
  </div>
);

const DLLSelectionModal: FC<{ gameName: string; closeModal?: () => void; selectDLL: () => void }> = ({
  gameName,
  closeModal,
  selectDLL
}) => (
  <ConfirmModal
    strTitle={<SetupTitle stage={dllSelectionStage} label='1 of 4 · Select Steam API DLL' />}
    strOKButtonText='Select Steam API DLL'
    strCancelButtonText='Close'
    onOK={selectDLL}
    onCancel={closeModal}
    onEscKeypress={closeModal}
  >
    <style>{setupModalStyles}</style>
    <DialogBody className='sentinel-gbe-setup-content'>
      <DialogBodyText>
        Sentinel uses the GBE Fork to set up achievements for <strong>{gameName || 'this game'}</strong>.
      </DialogBodyText>
      <DialogBodyText>
        Choose either steam_api64.dll or steam_api.dll from the game’s installation folder.
      </DialogBodyText>
    </DialogBody>
  </ConfirmModal>
);

const SetupModal: FC<{
  appId: string;
  gameName: string;
  dllPath: string;
  hasBackups: boolean;
  closeModal?: () => void;
}> = ({ appId, gameName, dllPath, hasBackups, closeModal }) => {
  const [state, setState] = useState<SetupState>({ kind: 'confirm', progressStage: dllSelectionStage });

  useEffect(() => {
    const listener = (next: GBESetupUpdate) => {
      setState((current) => applySetupUpdate(current, next));
    };

    return subscribeGBESetupUpdates(listener);
  }, []);

  const run = async () => {
    setState({ kind: 'running', progressStage: dllSelectionStage });
    try {
      await fetcher.post(`${BASE_URL}/gbe-setup/start`, {
        appId,
        dllPath,
        confirmExistingBackups: hasBackups
      });
    } catch (error) {
      setState((current) => {
        if (current.kind !== 'running') {
          return current;
        }

        return {
          kind: 'terminal',
          update: { phase: 'failed', message: String(error) },
          progressStage: current.progressStage
        };
      });
    }
  };

  const cancel = async () => {
    if (state.kind !== 'running') {
      return;
    }

    setState({ ...state, kind: 'cancelling' });
    try {
      const result = await fetcher.post<{ cancelled: boolean }>(`${BASE_URL}/gbe-setup/cancel`, {});
      if (!result.cancelled) {
        setState((current) => (current.kind === 'cancelling' ? { ...current, kind: 'running' } : current));
      }
    } catch (error) {
      setState((current) => {
        if (current.kind !== 'cancelling') {
          return current;
        }

        return {
          kind: 'terminal',
          update: { phase: 'failed', message: String(error) },
          progressStage: current.progressStage
        };
      });
    }
  };

  const handlePrimaryAction = () => {
    switch (state.kind) {
      case 'confirm':
        void run();
        return;
      case 'running':
        void cancel();
        return;
      case 'terminal':
        closeModal?.();
    }
  };

  const active = state.kind === 'running' || state.kind === 'cancelling';
  const update = state.kind === 'confirm' ? undefined : state.update;
  const progressStage = state.progressStage;
  const progressLabel = progressLabelFor(state);

  return (
    <ConfirmModal
      strTitle={
        <SetupTitle stage={progressStage} label={progressLabel} showProgress={showsSetupTitleProgress(state)} />
      }
      strOKButtonText={setupPrimaryActionLabel(state)}
      strCancelButtonText={state.kind === 'confirm' ? 'Back' : undefined}
      bAlertDialog={state.kind !== 'confirm'}
      bOKDisabled={state.kind === 'cancelling'}
      bDisableBackgroundDismiss={active}
      bHideCloseIcon={active}
      onOK={handlePrimaryAction}
      onCancel={() => {
        if (state.kind === 'confirm') {
          closeModal?.();
        }
      }}
      onEscKeypress={() => {
        if (!active) {
          closeModal?.();
        }
      }}
    >
      <style>{setupModalStyles}</style>
      <DialogBody className='sentinel-gbe-setup-content'>
        {state.kind === 'confirm' && (
          <>
            <DialogBodyText>Continuing will require approval in the Steam Mobile app.</DialogBodyText>
            {hasBackups && (
              <DialogBodyText style={{ color: '#f6c453' }}>
                Existing Sentinel backups will be preserved while the current setup is replaced.
              </DialogBodyText>
            )}
            {!hasBackups && (
              <DialogBodyText>
                Sentinel will preserve the current DLL and any current steam_settings before replacement.
              </DialogBodyText>
            )}
            <code style={{ overflowWrap: 'anywhere' }}>{dllPath}</code>
          </>
        )}
        {state.kind === 'running' && update?.phase === 'awaitingQr' && state.challengeURL && (
          <div className='sentinel-gbe-setup-qr'>
            <QRCode value={state.challengeURL} size={160} />
            <div className='sentinel-gbe-setup-qr-details'>
              <DialogBodyText className='sentinel-gbe-setup-qr-message'>
                Scan and approve with the Steam Mobile app.
              </DialogBodyText>
            </div>
          </div>
        )}
        {state.kind === 'running' && (update?.phase === 'generating' || update?.phase === 'installing') && (
          <DialogBodyText>
            Sentinel is preparing the achievement configuration and installing the required game files. This may take a
            moment.
          </DialogBodyText>
        )}
        {state.kind === 'terminal' && state.update.phase === 'cancelled' && (
          <DialogBodyText>
            Achievement setup was cancelled before installation. The game files were left unchanged.
          </DialogBodyText>
        )}
        {state.kind === 'terminal' && state.update.phase === 'timedOut' && state.challengeURL && (
          <div className='sentinel-gbe-setup-qr'>
            <QRCode value={state.challengeURL} size={160} blurred />
            <div className='sentinel-gbe-setup-qr-details'>
              <DialogBodyText className='sentinel-gbe-setup-qr-message'>{update?.message}</DialogBodyText>
            </div>
          </div>
        )}
        {state.kind === 'terminal' && state.update.phase === 'timedOut' && !state.challengeURL && (
          <DialogBodyText>{update?.message}</DialogBodyText>
        )}
        {state.kind === 'terminal' && state.update.phase === 'failed' && (
          <DialogBodyText>{update?.message ?? 'GBE setup failed.'}</DialogBodyText>
        )}
        {update?.phase === 'completed' && (
          <>
            <DialogBodyText>
              Achievement setup completed for <strong>{gameName || 'this game'}</strong>.
            </DialogBodyText>
            <DialogBodyText>GSE Tools: {update.gseVersion}</DialogBodyText>
            <DialogBodyText>GBE Fork DLL: {update.gbeVersion}</DialogBodyText>
          </>
        )}
      </DialogBody>
    </ConfirmModal>
  );
};

const UndoModal: FC<{ appId: string; gameName: string; closeModal?: () => void }> = ({
  appId,
  gameName,
  closeModal
}) => {
  const [state, setState] = useState<UndoState>({ kind: 'confirm' });

  useEffect(() => {
    const listener = (next: GBESetupUpdate) => {
      if (next.phase === 'undoCompleted') {
        setState({ kind: 'terminal', update: next });
      }
    };

    return subscribeGBESetupUpdates(listener);
  }, []);

  const restore = async () => {
    setState({ kind: 'restoring' });
    try {
      await fetcher.post(`${BASE_URL}/gbe-setup/${appId}/undo`, {});
    } catch (error) {
      setState((current) => {
        if (current.kind !== 'restoring') {
          return current;
        }

        return { kind: 'terminal', update: { phase: 'failed', message: String(error) } };
      });
    }
  };

  const finished = state.kind === 'terminal';
  const restoring = state.kind === 'restoring';
  const update = state.kind === 'terminal' ? state.update : undefined;

  return (
    <ConfirmModal
      modalClassName='sentinel-gbe-undo-modal'
      strTitle='Undo Achievements Setup'
      strOKButtonText={state.kind === 'confirm' ? 'Restore backups' : finished ? 'Close' : 'Restoring…'}
      strCancelButtonText={state.kind === 'confirm' ? 'Back' : undefined}
      bOKDisabled={restoring}
      bDisableBackgroundDismiss={restoring}
      bHideCloseIcon={restoring}
      onOK={() => (finished ? closeModal?.() : void restore())}
      onCancel={() => {
        if (!restoring) {
          closeModal?.();
        }
      }}
      onEscKeypress={() => {
        if (!restoring) {
          closeModal?.();
        }
      }}
    >
      <style>{setupModalStyles}</style>
      <div className='sentinel-gbe-undo-content'>
        {state.kind === 'confirm' && (
          <DialogBodyText>
            Restore the matching DLL and steam_settings backups for <strong>{gameName || 'this game'}</strong>?
          </DialogBodyText>
        )}
        {state.kind === 'restoring' && <DialogBodyText>Restoring Sentinel backups…</DialogBodyText>}
        {state.kind === 'terminal' && state.update.phase !== 'failed' && (
          <DialogBodyText>{update?.message ?? 'Achievement setup was undone.'}</DialogBodyText>
        )}
        {state.kind === 'terminal' && state.update.phase === 'failed' && (
          <DialogBodyText>{update?.message ?? 'GBE undo failed.'}</DialogBodyText>
        )}
      </div>
    </ConfirmModal>
  );
};

export function openGBESetup(appId: string, gameName: string) {
  let selectionModal: ReturnType<typeof showModal> | undefined;

  const selectDLL = async () => {
    selectionModal?.Close();
    try {
      const selected = await openFilePicker(
        FileSelectionType.FILE,
        '/home',
        true,
        true,
        undefined,
        ['dll'],
        false,
        true
      );
      if (!selected.realpath) {
        openGBESetup(appId, gameName);
        return;
      }

      const backupStatus = await fetcher.post<BackupStatus>(`${BASE_URL}/gbe-setup/preflight`, {
        appId,
        dllPath: selected.realpath
      });
      let setupModal: ReturnType<typeof showModal> | undefined;
      setupModal = showModal(
        <SetupModal
          appId={appId}
          gameName={gameName}
          dllPath={selected.realpath}
          hasBackups={backupStatus.hasDllBackup || backupStatus.hasSettingsBackup}
          closeModal={() => setupModal?.Close()}
        />
      );
    } catch (error) {
      showModal(<ConfirmModal strTitle='Setup Achievements unavailable' strDescription={String(error)} bAlertDialog />);
    }
  };

  selectionModal = showModal(
    <DLLSelectionModal
      gameName={gameName}
      closeModal={() => selectionModal?.Close()}
      selectDLL={() => void selectDLL()}
    />
  );
}

export function openGBEUndo(appId: string, gameName: string) {
  let modal: ReturnType<typeof showModal> | undefined;
  modal = showModal(<UndoModal appId={appId} gameName={gameName} closeModal={() => modal?.Close()} />);
}
