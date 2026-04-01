import type { FC } from 'react';
import { Outlet, ScrollRestoration } from 'react-router';
import { GamesProvider } from '@/shared/context/games-context';

const App: FC = () => {
  return (
    <GamesProvider>
      <Outlet />
      <ScrollRestoration />
    </GamesProvider>
  );
};
export default App;
