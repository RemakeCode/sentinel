import { showModal } from '@decky/ui';
import { AchievementSetupAction } from '@/shared/types/_generated/sentinel/backend/generator/models';
import type { Update } from '@/shared/types/_generated/sentinel/backend/generator/models';
import { GBESetupModal } from './gbe-setup-modal';
import { GBEUndoModal } from './gbe-undo-modal';

export { AchievementSetupAction };

export type GBESetupUpdateHandler = (update: Update) => void;
export type GBESetupUpdateSubscription = (handler: GBESetupUpdateHandler) => () => void;

const setupHandlers = new Set<GBESetupUpdateHandler>();

export const dispatchGBESetupUpdate = (update: Update) => {
  setupHandlers.forEach((handler) => handler(update));
};

export const subscribeGBESetupUpdates: GBESetupUpdateSubscription = (handler) => {
  setupHandlers.add(handler);
  return () => {
    setupHandlers.delete(handler);
  };
};

export const openAchievementSetup = (action: AchievementSetupAction, appId: string, gameName: string) => {
  let modal: ReturnType<typeof showModal> | undefined;
  const closeModal = () => modal?.Close();

  modal = showModal(
    action === AchievementSetupAction.AchievementSetupActionUndo ? (
      <GBEUndoModal appId={appId} gameName={gameName} closeModal={closeModal} />
    ) : (
      <GBESetupModal
        appId={appId}
        gameName={gameName}
        closeModal={closeModal}
        subscribeGBESetupUpdates={subscribeGBESetupUpdates}
      />
    )
  );
};
