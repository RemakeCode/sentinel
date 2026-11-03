export interface BackupStatus {
  id: string;
  operation: string;
  status: string;
  error?: string;
}

let latestStatus: BackupStatus | null = null;
const handlers = new Set<(status: BackupStatus) => void>();

export const getBackupStatus = () => latestStatus;

export const dispatchBackupStatus = (status: BackupStatus) => {
  latestStatus = status;
  handlers.forEach((handler) => handler(status));
};

export const subscribeBackupStatus = (handler: (status: BackupStatus) => void) => {
  handlers.add(handler);
  return () => {
    handlers.delete(handler);
  };
};
