import { createContext, useContext, useEffect, useState, type FC, type ReactNode } from 'react';
import { showToast } from '@/shared/components/toast';
import { Events } from '@wailsio/runtime';
import type { Status } from '@wa/sentinel/backend/backups/models';
import { useGames } from '@/shared/context/games-context';

interface BackupContextValue {
  status: Status | null;
}

const BackupContext = createContext<BackupContextValue>({ status: null });

export const useBackup = () => useContext(BackupContext);

export const BackupProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const [status, setStatus] = useState<Status | null>(null);
  const { refresh } = useGames();

  useEffect(() => {
    return Events.On('sentinel::backup-status', (event: { data: Status }) => {
      const next = event.data;
      setStatus(next);

      if (next.status === 'succeeded') {
        if (next.operation === 'restore') void refresh();
        showToast(next.operation === 'restore' ? 'Backup restored' : 'Backup created');
      } else if (next.status === 'failed') {
        showToast(next.error || 'Backup operation failed', 'danger');
      }
    });
  }, [refresh]);

  return <BackupContext.Provider value={{ status }}>{children}</BackupContext.Provider>;
};
