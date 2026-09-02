import './achievement-setup.scss';
import { useEffect, useRef, useState, type FC } from 'react';
import { renderSVG } from 'uqr';
import { CancelGBESetup, SelectTargetDLL, SetupGBE } from '@wa/sentinel/backend/generator/service';
import { Phase, SetupRequest, type Update } from '@wa/sentinel/backend/generator/models';

interface SetupProgressStage {
  value: number;
  label: string;
}

type SetupScreen = 'selection' | 'setup';

const dllSelectionStage: SetupProgressStage = { value: 15, label: 'Select Steam API DLL' };

function progressStageFor(phase: Phase): SetupProgressStage | undefined {
  switch (phase) {
    case Phase.PhasePreparing:
      return { value: 15, label: 'Preparing tools' };
    case Phase.PhaseDownloading:
      return { value: 25, label: 'Downloading tools' };
    case Phase.PhaseAwaitingQR:
      return { value: 35, label: 'Waiting for Steam approval' };
    case Phase.PhaseTimedOut:
      return { value: 35, label: 'Sign-in timed out' };
    case Phase.PhaseGenerating:
      return { value: 65, label: 'Generating configuration' };
    case Phase.PhaseInstalling:
      return { value: 85, label: 'Installing setup' };
    case Phase.PhaseCompleted:
      return { value: 100, label: 'Setup complete' };
    case Phase.PhaseCancelled:
      return { value: 0, label: 'Setup cancelled' };
    case Phase.PhaseFailed:
      return { value: 0, label: 'Setup failed' };
    default:
      return undefined;
  }
}

function isTerminalPhase(phase: Phase): boolean {
  return (
    phase === Phase.PhaseCompleted ||
    phase === Phase.PhaseFailed ||
    phase === Phase.PhaseCancelled ||
    phase === Phase.PhaseTimedOut
  );
}

function showsHeaderProgress(phase?: Phase): boolean {
  return phase !== Phase.PhaseCancelled && phase !== Phase.PhaseTimedOut;
}

function failedUpdate(message: string): Update {
  return { phase: Phase.PhaseFailed, message } as Update;
}

const SetupQRCode: FC<{ value: string; message: string; blurred?: boolean; label?: string }> = ({
  value,
  message,
  blurred = false,
  label = 'Steam sign-in QR code'
}) => (
  <div className='gbe-setup-dialog-qr'>
    <div className='gbe-setup-dialog-qr-code' role='img' aria-label={label}>
      <div
        className={`gbe-setup-dialog-qr-image${blurred ? ' gbe-setup-dialog-qr-image--blurred' : ''}`}
        dangerouslySetInnerHTML={{ __html: renderSVG(value, { border: 2 }) }}
      />
    </div>
    <div className='gbe-setup-dialog-qr-details'>
      <p className='gbe-setup-dialog-qr-message'>{message}</p>
    </div>
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
  request: { appId: string; gameName: string };
  dllPath: string;
  onSelected: (dllPath: string) => void;
  onContinue: () => void;
  onClose: () => void;
}> = ({ request, dllPath, onSelected, onContinue, onClose }) => {
  const selectDLL = async () => {
    try {
      const dllPath = await SelectTargetDLL();
      if (!dllPath) {
        return;
      }
      onSelected(dllPath);
    } catch (error) {
      console.error('Unable to open the Steam API DLL picker', error);
    }
  };

  return (
    <>
      <SetupHeader stage={dllSelectionStage} label={dllSelectionStage.label} />
      <div className='gbe-setup-dialog-content'>
        {!dllPath ? (
          <div className='gbe-setup-dialog-copy'>
            <p>
              Sentinel uses the GBE Fork to set up achievements for <strong>{request.gameName || 'this game'}</strong>.
            </p>
            <p>Choose either steam_api64.dll or steam_api.dll from the game’s installation folder.</p>
          </div>
        ) : (
          <div className='gbe-setup-dialog-copy'>
            <p>Continuing will require approval in the Steam Mobile app.</p>
            <p>Sentinel preserves the original DLL and any original steam_settings for Undo.</p>
            <code>{dllPath}</code>
          </div>
        )}
      </div>
      <footer>
        {!dllPath && (
          <button autoFocus onClick={() => void selectDLL()}>
            Select Steam API DLL
          </button>
        )}
        {dllPath && <button onClick={onContinue}>Continue</button>}
        <button className='outline' onClick={onClose}>
          Back
        </button>
      </footer>
    </>
  );
};

const SetupFlow: FC<{
  appId: string;
  dllPath: string;
  gameName: string;
  onActiveChange: (active: boolean) => void;
  onClose: () => void;
}> = ({ appId, dllPath, gameName, onActiveChange, onClose }) => {
  const [update, setUpdate] = useState<Update>({
    phase: Phase.PhasePreparing,
    message: 'Preparing required tools…'
  });
  const [cancelPending, setCancelPending] = useState(false);
  const startRequestedRef = useRef(false);

  useEffect(() => {
    const unsubscribe = Events.On('sentinel::achievement-setup-update', (event: { data: Update }) => {
      if (isTerminalPhase(event.data.phase)) {
        setCancelPending(false);
      }
      if (event.data.phase !== Phase.PhaseUndoCompleted) {
        setUpdate(event.data);
      }
    });

    return unsubscribe;
  }, []);

  useEffect(() => {
    if (startRequestedRef.current) {
      return;
    }

    startRequestedRef.current = true;
    setCancelPending(false);

    void SetupGBE(new SetupRequest({ appId, gameName, dllPath })).catch((error) => {
      setUpdate((current) => {
        if (current.phase !== Phase.PhasePreparing && current.phase !== Phase.PhaseDownloading) {
          return current;
        }
        return failedUpdate(String(error));
      });
    });
  }, [appId, gameName, dllPath]);

  const active = !isTerminalPhase(update.phase);

  useEffect(() => onActiveChange(active), [active, onActiveChange]);

  const cancelSetup = async () => {
    if (!active || cancelPending) {
      return;
    }

    setCancelPending(true);
    try {
      if (!(await CancelGBESetup())) {
        setCancelPending(false);
      }
    } catch (error) {
      setCancelPending(false);
      setUpdate((current) => {
        if (!current || isTerminalPhase(current.phase)) {
          return current;
        }

        return failedUpdate(String(error));
      });
    }
  };

  const progressStage = progressStageFor(update.phase) ?? dllSelectionStage;
  const progressLabel = cancelPending ? 'Cancelling setup' : progressStage.label;

  return (
    <>
      <SetupHeader stage={progressStage} label={progressLabel} showProgress={showsHeaderProgress(update.phase)} />
      <div className='gbe-setup-dialog-content'>
        {update.phase === Phase.PhaseAwaitingQR && update.challengeUrl && (
          <SetupQRCode value={update.challengeUrl} message='Scan and approve with the Steam Mobile app.' />
        )}
        {(update.phase === Phase.PhasePreparing || update.phase === Phase.PhaseDownloading) && (
          <p>{update.message ?? 'Preparing required tools…'}</p>
        )}
        {(update.phase === Phase.PhaseGenerating || update.phase === Phase.PhaseInstalling) && (
          <p>
            Sentinel is preparing the achievement configuration and installing the required game files. This may take a
            moment.
          </p>
        )}
        {update.phase === Phase.PhaseCancelled && (
          <p>Achievement setup was cancelled before installation. The game files were left unchanged.</p>
        )}
        {update.phase === Phase.PhaseTimedOut && (
          <SetupQRCode
            value='Hello World'
            message={update.message ?? 'Steam sign-in timed out.'}
            blurred
            label='Expired Steam sign-in QR code'
          />
        )}
        {update.phase === Phase.PhaseFailed && (
          <p className='gbe-setup-dialog-error'>{update.message ?? 'GBE setup failed.'}</p>
        )}
        {update.phase === Phase.PhaseCompleted && (
          <div className='gbe-setup-dialog-copy'>
            <p>
              Achievement setup completed for <strong>{gameName || 'this game'}</strong>.
            </p>
            <p>GSE Tools: {update.gseVersion}</p>
            <p>GBE Fork DLL: {update.gbeVersion}</p>
          </div>
        )}
      </div>
      <footer>
        {active && (
          <button onClick={() => void cancelSetup()} disabled={cancelPending}>
            Cancel
          </button>
        )}
        {!active && <button onClick={onClose}>Close</button>}
      </footer>
    </>
  );
};

export const GBESetupModal: FC<{
  isOpen: boolean;
  appId: string;
  gameName: string;
  onClose: () => void;
}> = ({ isOpen, appId, gameName, onClose }) => {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [screen, setScreen] = useState<SetupScreen>('selection');
  const [dllPath, setDLLPath] = useState('');
  const [active, setActive] = useState(false);

  useEffect(() => {
    setScreen('selection');
    setDLLPath('');
    setActive(false);
  }, [appId, gameName, isOpen]);

  useEffect(() => {
    if (isOpen && !dialogRef.current?.open) {
      dialogRef.current?.showModal();
    }
    if (!isOpen && dialogRef.current?.open) {
      dialogRef.current.close();
    }
  }, [isOpen]);

  const request = { appId, gameName };

  return (
    <dialog
      ref={dialogRef}
      className='gbe-setup-dialog'
      onCancel={(event) => {
        event.preventDefault();
        if (!active) {
          onClose();
        }
      }}
    >
      {screen === 'selection' && (
        <DLLSelectionFlow
          request={request}
          dllPath={dllPath}
          onSelected={(selectedDLLPath) => {
            setDLLPath(selectedDLLPath);
          }}
          onContinue={() => setScreen('setup')}
          onClose={onClose}
        />
      )}
      {screen === 'setup' && (
        <SetupFlow
          key={`${appId}:${dllPath}`}
          appId={request.appId}
          dllPath={dllPath}
          gameName={request.gameName}
          onActiveChange={setActive}
          onClose={onClose}
        />
      )}
    </dialog>
  );
};
