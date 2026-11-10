import { createContext, useContext, useEffect, useRef, useState, type FC, type ReactNode } from 'react';
import { Events } from '@wailsio/runtime';
import { Check } from '@wa/sentinel/backend/updater/service';

interface UpdateRelease {
  version: string;
  notes?: string;
}

interface UpdateContextValue {
  release: UpdateRelease | null;
  checking: boolean;
  error: string;
  check: () => Promise<void>;
}

const UpdateContext = createContext<UpdateContextValue | null>(null);

export const useUpdate = () => {
  const value = useContext(UpdateContext);
  if (!value) throw new Error('useUpdate requires UpdateProvider');
  return value;
};

export const UpdateProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const [release, setRelease] = useState<UpdateRelease | null>(null);
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState('');
  const started = useRef(false);

  const check = async () => {
    setChecking(true);
    setError('');
    try {
      await Check();
    } catch {
      setError('Unable to find update');
    } finally {
      setChecking(false);
    }
  };

  useEffect(() => {
    const offAvailable = Events.On('wails:updater:update-available', (event: { data: UpdateRelease }) => {
      setRelease(event.data);
    });
    const offNoUpdate = Events.On('wails:updater:no-update', () => setRelease(null));

    // Listen before the startup check. StrictMode must not start it twice.
    if (!started.current) {
      started.current = true;
      void check();
    }

    return () => { offAvailable(); offNoUpdate(); };
  }, []);

  return (
    <UpdateContext.Provider value={{ release, checking, error, check }}>
      {children}
    </UpdateContext.Provider>
  );
};
