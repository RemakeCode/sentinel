import './app.scss';
import type { FC } from 'react';
import { useEffect, useState } from 'react';
import { Outlet, ScrollRestoration } from 'react-router';
import { GetAppInfo } from '@wa/sentinel/backend/config/file';
import { GamesProvider } from '@/shared/context/games-context';


const App: FC = () => {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    GetAppInfo().then(() => setReady(true)).catch(() => setReady(true));
  }, []);

  if (!ready) {
    return (
      <div className="app-loader">
        <div className="hstack">
          <div aria-busy="true" data-spinner="large" />
          <span>Loading Sentinel ...</span>
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
