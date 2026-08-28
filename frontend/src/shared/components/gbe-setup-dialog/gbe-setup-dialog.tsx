import './gbe-setup-dialog.scss';
import { useEffect, useRef, useState, type FC } from 'react';
import { Events } from '@wailsio/runtime';
import { renderSVG } from 'uqr';
import {
  CancelGBESetup,
  SelectTargetDLL,
  SetupGBE,
  UndoGBESetup
} from '@wa/sentinel/backend/generator/service';
import { Phase, SetupRequest, type Update } from '@wa/sentinel/backend/generator/models';

interface SetupTarget {
  appId: string;
  dllPath: string;
}

interface SetupDialogRequest {
  appId: string;
  gameName: string;
}

interface UndoDialogRequest {
  appId: string;
  gameName: string;
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
      update?: Update;
      progressStage: SetupProgressStage;
      challengeURL?: string;
    }
  | {
      kind: 'terminal';
      update: Update;
      progressStage: SetupProgressStage;
      challengeURL?: string;
    };

type UndoState = { kind: 'confirm' } | { kind: 'restoring' } | { kind: 'terminal'; update: Update };

type DialogState =
  | { kind: 'closed' }
  | { kind: 'selectDLL'; request: SetupDialogRequest }
  | { kind: 'setup'; gameName: string; target: SetupTarget }
  | { kind: 'failure'; message: string }
  | { kind: 'undo'; request: UndoDialogRequest };

function progressStageFor(phase: Phase): SetupProgressStage | undefined {
  switch (phase) {
    case Phase.PhasePreparing:
      return { value: 15, step: 1, label: 'Preparing tools' };
    case Phase.PhaseAwaitingQR:
      return { value: 35, step: 2, label: 'Waiting for Steam approval' };
    case Phase.PhaseGenerating:
      return { value: 65, step: 3, label: 'Generating configuration' };
    case Phase.PhaseInstalling:
      return { value: 85, step: 4, label: 'Installing setup' };
    case Phase.PhaseCompleted:
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
        case Phase.PhaseCompleted:
          return 'Setup complete';
        case Phase.PhaseCancelled:
          return 'Setup cancelled';
        case Phase.PhaseTimedOut:
          return 'Sign-in timed out';
        case Phase.PhaseFailed:
          return 'Setup failed';
      }
  }

  return `${state.progressStage.step} of 4 · ${state.progressStage.label}`;
}

function showsHeaderProgress(state: SetupState): boolean {
  return (
    state.kind !== 'terminal' ||
    (state.update.phase !== Phase.PhaseCancelled && state.update.phase !== Phase.PhaseTimedOut)
  );
}

function failedUpdate(message: string): Update {
  return { phase: Phase.PhaseFailed, message } as Update;
}

function applySetupUpdate(state: SetupState, update: Update): SetupState {
  if (update.phase === Phase.PhaseUndoCompleted) {
    return state;
  }

  const progressStage = progressStageFor(update.phase) ?? state.progressStage;
  const challengeURL = update.challengeUrl ?? ('challengeURL' in state ? state.challengeURL : undefined);

  if (
    update.phase === Phase.PhaseCompleted ||
    update.phase === Phase.PhaseFailed ||
    update.phase === Phase.PhaseCancelled ||
    update.phase === Phase.PhaseTimedOut
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

const QRCode: FC<{ value: string; blurred?: boolean }> = ({ value, blurred = false }) => (
  <div className='gbe-setup-dialog-qr-code' role='img' aria-label='Steam sign-in QR code'>
    <div
      className={`gbe-setup-dialog-qr-image${blurred ? ' gbe-setup-dialog-qr-image--blurred' : ''}`}
      dangerouslySetInnerHTML={{ __html: renderSVG(value, { border: 2 }) }}
    />
  </div>
);

const SetupProgress: FC<{ stage?: SetupProgressStage; label: string }> = ({ stage, label }) => (
  <div className='gbe-setup-dialog-progress'>
    <span>{label}</span>
    <progress value={stage?.value ?? 0} max={100} />
  </div>
);

const SetupHeader: FC<{ stage: SetupProgressStage; label: string; showProgress?: boolean }> = ({
  stage,
  label,
  showProgress = true
}) => (
  <header className='gbe-setup-dialog-header'>
    <h3>Setup Achievements</h3>
    {showProgress && <SetupProgress stage={stage} label={label} />}
  </header>
);

const DLLSelectionFlow: FC<{
  request: SetupDialogRequest;
  onSelected: (target: SetupTarget) => void;
  onFailure: (message: string) => void;
  onClose: () => void;
}> = ({ request, onSelected, onFailure, onClose }) => {
  const selectDLL = async () => {
    try {
      const dllPath = await SelectTargetDLL();
      if (!dllPath) {
        return;
      }

      onSelected({
        appId: request.appId,
        dllPath
      });
    } catch (error) {
      onFailure(String(error));
    }
  };

  return (
    <>
      <SetupHeader stage={dllSelectionStage} label='1 of 4 · Select Steam API DLL' />
      <div className='gbe-setup-dialog-content'>
        <div className='gbe-setup-dialog-copy'>
          <p>
            Sentinel uses the GBE Fork to set up achievements for <strong>{request.gameName || 'this game'}</strong>.
          </p>
          <p>Choose either steam_api64.dll or steam_api.dll from the game’s installation folder.</p>
        </div>
      </div>
      <footer>
        <button autoFocus onClick={() => void selectDLL()}>
          Select Steam API DLL
        </button>
        <button className='outline' onClick={onClose}>
          Close
        </button>
      </footer>
    </>
  );
};

const SetupFlow: FC<{
  gameName: string;
  target: SetupTarget;
  onActiveChange: (active: boolean) => void;
  onClose: () => void;
}> = ({ gameName, target, onActiveChange, onClose }) => {
  const [state, setState] = useState<SetupState>({ kind: 'confirm', progressStage: dllSelectionStage });

  useEffect(() => {
    const off = Events.On('sentinel::gbe-setup', (event: { data: Update }) => {
      setState((current) => applySetupUpdate(current, event.data));
    });

    return off;
  }, []);

  const active = state.kind === 'running' || state.kind === 'cancelling';
  useEffect(() => onActiveChange(active), [active, onActiveChange]);

  const startSetup = async () => {
    if (state.kind !== 'confirm') {
      return;
    }

    setState({ kind: 'running', progressStage: dllSelectionStage });
    try {
      await SetupGBE(new SetupRequest(target));
    } catch (error) {
      setState((current) => {
        if (current.kind !== 'running') {
          return current;
        }

        return {
          kind: 'terminal',
          update: failedUpdate(String(error)),
          progressStage: current.progressStage
        };
      });
    }
  };

  const cancelSetup = async () => {
    if (state.kind !== 'running') {
      return;
    }

    setState({ ...state, kind: 'cancelling' });
    try {
      if (!(await CancelGBESetup())) {
        setState((current) => (current.kind === 'cancelling' ? { ...current, kind: 'running' } : current));
      }
    } catch (error) {
      setState((current) => {
        if (current.kind !== 'cancelling') {
          return current;
        }

        return {
          kind: 'terminal',
          update: failedUpdate(String(error)),
          progressStage: current.progressStage
        };
      });
    }
  };

  return (
    <>
      <SetupHeader
        stage={state.progressStage}
        label={progressLabelFor(state)}
        showProgress={showsHeaderProgress(state)}
      />
      <div className='gbe-setup-dialog-content'>
        {state.kind === 'confirm' && (
          <div className='gbe-setup-dialog-copy'>
            <p>Continuing will require approval in the Steam Mobile app.</p>
            <p>Sentinel preserves the original DLL and any original steam_settings for Undo.</p>
            <code>{target.dllPath}</code>
          </div>
        )}
        {state.kind === 'running' && state.update?.phase === Phase.PhaseAwaitingQR && state.challengeURL && (
          <div className='gbe-setup-dialog-qr'>
            <QRCode value={state.challengeURL} />
            <div className='gbe-setup-dialog-qr-details'>
              <p className='gbe-setup-dialog-qr-message'>Scan and approve with the Steam Mobile app.</p>
            </div>
          </div>
        )}
        {state.kind === 'running' &&
          (state.update?.phase === Phase.PhaseGenerating || state.update?.phase === Phase.PhaseInstalling) && (
            <p>
              Sentinel is preparing the achievement configuration and installing the required game files. This may take
              a moment.
            </p>
          )}
        {state.kind === 'terminal' && state.update.phase === Phase.PhaseCancelled && (
          <p>Achievement setup was cancelled before installation. The game files were left unchanged.</p>
        )}
        {state.kind === 'terminal' && state.update.phase === Phase.PhaseTimedOut && state.challengeURL && (
          <div className='gbe-setup-dialog-qr'>
            <QRCode value={state.challengeURL} blurred />
            <p className='gbe-setup-dialog-qr-message'>{state.update.message}</p>
          </div>
        )}
        {state.kind === 'terminal' && state.update.phase === Phase.PhaseTimedOut && !state.challengeURL && (
          <p>{state.update.message}</p>
        )}
        {state.kind === 'terminal' && state.update.phase === Phase.PhaseFailed && (
          <p className='gbe-setup-dialog-error'>{state.update.message ?? 'GBE setup failed.'}</p>
        )}
        {state.kind === 'terminal' && state.update.phase === Phase.PhaseCompleted && (
          <div className='gbe-setup-dialog-copy'>
            <p>
              Achievement setup completed for <strong>{gameName || 'this game'}</strong>.
            </p>
            <p>GSE Tools: {state.update.gseVersion}</p>
            <p>GBE Fork DLL: {state.update.gbeVersion}</p>
          </div>
        )}
      </div>
      <footer>
        {state.kind === 'confirm' && <button onClick={() => void startSetup()}>Continue</button>}
        {state.kind === 'running' && <button onClick={() => void cancelSetup()}>Cancel</button>}
        {state.kind === 'cancelling' && <button disabled>Cancelling…</button>}
        {state.kind === 'terminal' && <button onClick={onClose}>Close</button>}
        {state.kind === 'confirm' && (
          <button className='outline' onClick={onClose}>
            Back
          </button>
        )}
      </footer>
    </>
  );
};

const UndoFlow: FC<{
  request: UndoDialogRequest;
  onActiveChange: (active: boolean) => void;
  onClose: () => void;
}> = ({ request, onActiveChange, onClose }) => {
  const [state, setState] = useState<UndoState>({ kind: 'confirm' });

  useEffect(() => {
    const off = Events.On('sentinel::gbe-setup', (event: { data: Update }) => {
      if (event.data.phase === Phase.PhaseUndoCompleted) {
        setState({ kind: 'terminal', update: event.data });
      }
    });

    return off;
  }, []);

  const restoring = state.kind === 'restoring';
  useEffect(() => onActiveChange(restoring), [onActiveChange, restoring]);

  const restoreBackups = async () => {
    if (state.kind !== 'confirm') {
      return;
    }

    setState({ kind: 'restoring' });
    try {
      await UndoGBESetup(request.appId);
    } catch (error) {
      setState((current) => {
        if (current.kind !== 'restoring') {
          return current;
        }

        return { kind: 'terminal', update: failedUpdate(String(error)) };
      });
    }
  };

  return (
    <>
      <header className='gbe-setup-dialog-header gbe-setup-dialog-header--without-progress'>
        <h3>Undo Achievement Setup</h3>
      </header>
      <div className='gbe-setup-dialog-content'>
        {state.kind === 'confirm' && (
          <p>
            Restore the matching DLL and steam_settings backups for <strong>{request.gameName || 'this game'}</strong>?
          </p>
        )}
        {state.kind === 'restoring' && <p>Restoring Sentinel backups…</p>}
        {state.kind === 'terminal' && state.update.phase === Phase.PhaseFailed && (
          <p className='gbe-setup-dialog-error'>{state.update.message ?? 'GBE undo failed.'}</p>
        )}
        {state.kind === 'terminal' && state.update.phase !== Phase.PhaseFailed && (
          <p>{state.update.message ?? 'Achievement setup was undone.'}</p>
        )}
      </div>
      <footer>
        {state.kind === 'confirm' && <button onClick={() => void restoreBackups()}>Restore backups</button>}
        {state.kind === 'restoring' && <button disabled>Restoring…</button>}
        {state.kind === 'terminal' && <button onClick={onClose}>Close</button>}
        {state.kind === 'confirm' && (
          <button className='outline' onClick={onClose}>
            Back
          </button>
        )}
      </footer>
    </>
  );
};

const FailureFlow: FC<{ message: string; onClose: () => void }> = ({ message, onClose }) => (
  <>
    <header>
      <h3>Setup Achievements unavailable</h3>
    </header>
    <div className='gbe-setup-dialog-content gbe-setup-dialog-content--failure'>
      <p className='gbe-setup-dialog-error'>{message}</p>
    </div>
    <footer>
      <button autoFocus onClick={onClose}>
        Close
      </button>
    </footer>
  </>
);

export const GBESetupDialog: FC = () => {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [dialog, setDialog] = useState<DialogState>({ kind: 'closed' });
  const [active, setActive] = useState(false);

  const dismiss = () => {
    setActive(false);
    setDialog({ kind: 'closed' });
  };

  useEffect(() => {
    const setupOff = Events.On('sentinel::gbe-setup-requested', (event: { data: SetupDialogRequest }) => {
      setActive(false);
      setDialog({ kind: 'selectDLL', request: event.data });
    });
    const undoOff = Events.On('sentinel::gbe-undo-requested', (event: { data: UndoDialogRequest }) => {
      setActive(false);
      setDialog({ kind: 'undo', request: event.data });
    });

    return () => {
      setupOff();
      undoOff();
    };
  }, []);

  useEffect(() => {
    if (dialog.kind !== 'closed' && !dialogRef.current?.open) {
      dialogRef.current?.showModal();
    }
  }, [dialog.kind]);

  if (dialog.kind === 'closed') {
    return null;
  }

  return (
    <dialog
      ref={dialogRef}
      className='gbe-setup-dialog'
      onClose={dismiss}
      onCancel={(event) => {
        event.preventDefault();
        if (!active) {
          dismiss();
        }
      }}
    >
      {dialog.kind === 'selectDLL' && (
        <DLLSelectionFlow
          request={dialog.request}
          onSelected={(target) => setDialog({ kind: 'setup', gameName: dialog.request.gameName, target })}
          onFailure={(message) => setDialog({ kind: 'failure', message })}
          onClose={dismiss}
        />
      )}
      {dialog.kind === 'setup' && (
        <SetupFlow gameName={dialog.gameName} target={dialog.target} onActiveChange={setActive} onClose={dismiss} />
      )}
      {dialog.kind === 'undo' && <UndoFlow request={dialog.request} onActiveChange={setActive} onClose={dismiss} />}
      {dialog.kind === 'failure' && <FailureFlow message={dialog.message} onClose={dismiss} />}
    </dialog>
  );
};
