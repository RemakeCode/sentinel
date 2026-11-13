import { BASE_URL, Fetcher } from '@/shared/utils/fetcher';

export interface UpdateRelease {
  version: string;
  notes: string;
  url: string;
  size: number;
  sha256: string;
}

export interface PreparedUpdate {
  path: string;
  version: string;
  sha256: string;
}

export interface UpdateState {
  phase: string;
  release?: UpdateRelease;
  prepared?: PreparedUpdate;
  downloaded: number;
  total: number;
  error?: string;
}

let latestState: UpdateState = { phase: 'idle', downloaded: 0, total: 0 };
const handlers = new Set<(state: UpdateState) => void>();
const fetcher = new Fetcher(`${BASE_URL}/updates`);

export const getUpdateStatus = () => latestState;
export const dispatchUpdateStatus = (state: UpdateState) => {
  latestState = state;
  handlers.forEach((handler) => handler(state));
};
export const subscribeUpdateStatus = (handler: (state: UpdateState) => void) => {
  handlers.add(handler);
  return () => { handlers.delete(handler); };
};
export const checkForUpdate = async () => {
  const state = await fetcher.post<UpdateState>('/check', {});
  dispatchUpdateStatus(state);
  return state;
};
export const prepareUpdate = (version: string) => fetcher.post<UpdateState>('/prepare', { version });
