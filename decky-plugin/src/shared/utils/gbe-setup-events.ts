import type { Update } from '@/shared/types/_generated/sentinel/backend/generator/models';

export type GBESetupUpdateHandler = (update: Update) => void;
export type GBESetupUpdateSubscription = (handler: GBESetupUpdateHandler) => () => void;

const setupHandlers = new Set<GBESetupUpdateHandler>();

export const dispatchGBESetupUpdate = (update: Update) => {
  setupHandlers.forEach((handler) => handler(update));
};

export const subscribeGBESetupUpdates: GBESetupUpdateSubscription = (handler: GBESetupUpdateHandler) => {
  setupHandlers.add(handler);
  return () => {
    setupHandlers.delete(handler);
  };
};
