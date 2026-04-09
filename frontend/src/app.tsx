import type { FC } from 'react';
import { useEffect, useState } from 'react';
import { Outlet, ScrollRestoration } from 'react-router';
import { GetAppInfo } from '@wa/sentinel/backend/config/file';
import { GamesProvider } from '@/shared/context/games-context';

import './app.scss';

const App: FC = () => {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    GetAppInfo().then(() => setReady(true)).catch(() => setReady(true));
  }, []);

  if (!ready) {
    return (
      <div class="app-loader">
        <div class="app-loader-content">
          <div aria-busy="true" data-spinner="large" />
          <p class="app-loader-text">Loading Sentinel...</p>
        </div>
      </div>
    );
  }

  return (
    <GamesProvider>
      <Outlet />
      <ScrollRestoration />
    </GamesProvider>
  );
};

export default App;