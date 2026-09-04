import { type FC, useEffect, useRef, useState } from 'react';
import { FileSelectionType, openFilePicker } from '@decky/api';
import { ConfirmModal, DialogBody, DialogBodyText, ProgressBar } from '@decky/ui';
import { renderSVG } from 'uqr';
import trophyOverlay from '../../../../assets/qr-overlay-trophy.png';
import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';
import { sentinelLogger } from '@/shared/utils/logger';
import type { Update } from '@/shared/types/_generated/sentinel/backend/generator/models';
import { Phase } from '@/shared/types/_generated/sentinel/backend/generator/models';
import { achievementSetupStyles } from '@/pages/settings/achievement-setup/achievement-setup-styles';
import type { GBESetupUpdateSubscription } from '@/shared/utils/gbe-setup-events';

interface SetupProgressStage {
  value: number;
  label: string;
}

type SetupScreen = 'selection' | 'setup';

const dllSelectionStage: SetupProgressStage = { value: 5, label: 'Select Steam API DLL' };

function isSteamApiDLL(path: string): boolean {
  return path.endsWith('/steam_api.dll') || path.endsWith('/steam_api64.dll');
}

const preparingUpdate: Update = {
  phase: Phase.PhasePreparing,
  message: 'Preparing required tools…'
};

const fetcher = new Fetcher();

function progressStageFor(phase: Phase): SetupProgressStage | undefined {
  switch (phase) {
    case Phase.PhasePreparing:
      return { value: 15, label: 'Preparing tools' };
    case Phase.PhaseDownloading:
      return { value: 25, label: 'Downloading tools' };
    case Phase.PhaseAwaitingQR:
      return { value: 35, label: 'Waiting for Steam approval' };
    case Phase.PhaseGenerating:
      return { value: 65, label: 'Generating configuration' };
    case Phase.PhaseInstalling:
      return { value: 85, label: 'Installing setup' };
    case Phase.PhaseCompleted:
      return { value: 100, label: 'Setup complete' };
    case Phase.PhaseCancelled:
      return { value: 0, label: 'Setup cancelled' };
    case Phase.PhaseTimedOut:
      return { value: 35, label: 'Sign-in timed out' };
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

function showsSetupTitleProgress(phase: Phase): boolean {
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
  <div className='sentinel-gbe-setup-qr'>
    <div className='sentinel-gbe-setup-qr-code' role='img' aria-label={label}>
      <div
        className={`sentinel-gbe-setup-qr-image${blurred ? ' sentinel-gbe-setup-qr-image--blurred' : ''}`}
      >
        <div dangerouslySetInnerHTML={{ __html: renderSVG(value, { border: 2, ecc: 'H' }) }} />
        <img className='sentinel-gbe-setup-qr-overlay' src={trophyOverlay} alt='' />
      </div>
    </div>
    <div className='sentinel-gbe-setup-qr-details'>
      <DialogBodyText className='sentinel-gbe-setup-qr-message'>{message}</DialogBodyText>
    </div>
  </div>
);

const SetupTitle: FC<{ stage: SetupProgressStage; label: string; showProgress?: boolean }> = ({
  stage,
  label,
  showProgress = true
}) => (
  <div className='sentinel-gbe-setup-title'>
    <span className='sentinel-gbe-setup-title-text'>Setup Achievements</span>
    {showProgress && (
      <div className='sentinel-gbe-setup-progress'>
        <span>{label}</span>
        <ProgressBar nProgress={stage.value} focusable={false} />
      </div>
    )}
  </div>
);

const DLLSelectionFlow: FC<{
  gameName: string;
  dllPath: string;
  onSelected: (dllPath: string) => void;
  onContinue: () => void;
  onClose: () => void;
}> = ({ gameName, dllPath, onSelected, onContinue, onClose }) => {
  const [selectionError, setSelectionError] = useState(false);

  const selectDLL = async () => {
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
      if (selected.realpath) {
        if (!isSteamApiDLL(selected.realpath)) {
          setSelectionError(true);
          return;
        }
        setSelectionError(false);
        onSelected(selected.realpath);
      }
    } catch (error) {
      sentinelLogger.error('Unable to open the Steam API DLL picker', error);
    }
  };

  return (
    <ConfirmModal
      strTitle={<SetupTitle stage={dllSelectionStage} label={dllSelectionStage.label} />}
      strOKButtonText={dllPath ? 'Continue' : 'Select Steam API DLL'}
      strCancelButtonText='Back'
      onOK={() => (dllPath ? onContinue() : void selectDLL())}
      onCancel={onClose}
      onEscKeypress={onClose}
    >
      <style>{achievementSetupStyles}</style>
      <DialogBody className='sentinel-gbe-setup-content'>
        {!dllPath ? (
          <>
            <DialogBodyText>
              Sentinel uses the GBE Fork to set up achievements for <strong>{gameName || 'this game'}</strong>.
            </DialogBodyText>
            <DialogBodyText>
              Choose either steam_api64.dll or steam_api.dll from the game’s installation folder.
            </DialogBodyText>
            {selectionError && (
              <DialogBodyText className='sentinel-gbe-setup-error'>
                Select either steam_api.dll or steam_api64.dll from the game folder.
              </DialogBodyText>
            )}
          </>
        ) : (
          <>
            <DialogBodyText>Continuing will require approval in the Steam Mobile app.</DialogBodyText>
            <DialogBodyText>
              Sentinel preserves the original DLL and any original steam_settings for Undo.
            </DialogBodyText>
            <code style={{ overflowWrap: 'anywhere' }}>{dllPath}</code>
          </>
        )}
      </DialogBody>
    </ConfirmModal>
  );
};

const SetupFlow: FC<{
  appId: string;
  gameName: string;
  dllPath: string;
  closeModal: () => void;
  onSetupTerminal: () => void;
  subscribeGBESetupUpdates: GBESetupUpdateSubscription;
}> = ({ appId, gameName, dllPath, closeModal, onSetupTerminal, subscribeGBESetupUpdates }) => {
  const [update, setUpdate] = useState<Update>(preparingUpdate);
  const [cancelPending, setCancelPending] = useState(false);
  const startRequestedRef = useRef(false);

  useEffect(
    () =>
      subscribeGBESetupUpdates((next) => {
        if (next.phase === Phase.PhaseUndoCompleted) {
          return;
        }
        if (isTerminalPhase(next.phase)) {
          setCancelPending(false);
          onSetupTerminal();
        }
        setUpdate(next);
      }),
    [subscribeGBESetupUpdates]
  );

  useEffect(() => {
    if (startRequestedRef.current) {
      return;
    }

    startRequestedRef.current = true;
    void fetcher.post(`${BASE_URL}/gbe-setup/start`, { appId, gameName, dllPath }).catch((error) => {
      onSetupTerminal();
      setUpdate((current) => {
        if (isTerminalPhase(current.phase)) {
          return current;
        }
        return failedUpdate(String(error));
      });
    });
  }, [appId, gameName, dllPath]);

  const active = !isTerminalPhase(update.phase);

  const cancelSetup = async () => {
    if (!active || cancelPending) {
      return;
    }

    setCancelPending(true);
    try {
      const result = await fetcher.post<{ cancelled: boolean }>(`${BASE_URL}/gbe-setup/cancel`, {});
      if (!result.cancelled) {
        setCancelPending(false);
      }
    } catch {
      setCancelPending(false);
      onSetupTerminal();
      setUpdate((current) => {
        if (isTerminalPhase(current.phase)) {
          return current;
        }
        return failedUpdate('Unable to cancel achievement setup. Please try again.');
      });
    }
  };

  const progressStage = progressStageFor(update.phase) ?? dllSelectionStage;
  const progressLabel = cancelPending ? 'Cancelling setup' : progressStage.label;

  return (
    <ConfirmModal
      strTitle={
        <SetupTitle stage={progressStage} label={progressLabel} showProgress={showsSetupTitleProgress(update.phase)} />
      }
      strOKButtonText={active ? 'Cancel' : 'Close'}
      bAlertDialog
      bOKDisabled={cancelPending}
      bDisableBackgroundDismiss={active}
      bHideCloseIcon={active}
      onOK={() => (active ? void cancelSetup() : closeModal())}
      onCancel={() => {
        if (!active) {
          closeModal();
        }
      }}
      onEscKeypress={() => {
        if (!active) {
          closeModal();
        }
      }}
    >
      <style>{achievementSetupStyles}</style>
      <DialogBody className='sentinel-gbe-setup-content'>
        {update.phase === Phase.PhaseAwaitingQR && update.challengeUrl && (
          <SetupQRCode value={update.challengeUrl} message='Scan and approve with the Steam Mobile app.' />
        )}
        {(update.phase === Phase.PhasePreparing || update.phase === Phase.PhaseDownloading) && (
          <DialogBodyText>{update.message ?? 'Preparing required tools…'}</DialogBodyText>
        )}
        {(update.phase === Phase.PhaseGenerating || update.phase === Phase.PhaseInstalling) && (
          <DialogBodyText>
            Sentinel is preparing the achievement configuration and installing the required game files. This may take a
            moment.
          </DialogBodyText>
        )}
        {update.phase === Phase.PhaseCancelled && (
          <DialogBodyText>
            Achievement setup was cancelled before installation. The game files were left unchanged.
          </DialogBodyText>
        )}
        {update.phase === Phase.PhaseTimedOut && (
          <SetupQRCode
            value='Hello World'
            message={update.message ?? 'Steam sign-in timed out.'}
            blurred
            label='Expired Steam sign-in QR code'
          />
        )}
        {update.phase === Phase.PhaseFailed && <DialogBodyText>Achievement setup failed. Please try again.</DialogBodyText>}
        {update.phase === Phase.PhaseCompleted && (
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

export const GBESetupModal: FC<{
  appId: string;
  gameName: string;
  closeModal: () => void;
  onSetupTerminal: () => void;
  subscribeGBESetupUpdates: GBESetupUpdateSubscription;
}> = ({ appId, gameName, closeModal, onSetupTerminal, subscribeGBESetupUpdates }) => {
  const [screen, setScreen] = useState<SetupScreen>('selection');
  const [dllPath, setDLLPath] = useState('');

  if (screen === 'selection') {
    return (
      <DLLSelectionFlow
        gameName={gameName}
        dllPath={dllPath}
        onSelected={setDLLPath}
        onContinue={() => setScreen('setup')}
        onClose={closeModal}
      />
    );
  }

  return (
    <SetupFlow
      appId={appId}
      gameName={gameName}
      dllPath={dllPath}
      closeModal={closeModal}
      onSetupTerminal={onSetupTerminal}
      subscribeGBESetupUpdates={subscribeGBESetupUpdates}
    />
  );
};
