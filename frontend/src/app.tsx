import '@wailsio/runtime';

import { Outlet, ScrollRestoration } from 'react-router';
import { GamesProvider } from '@/shared/context/games-context';

const App = () => {
  return (
    <GamesProvider>
      <Outlet />
      <ScrollRestoration />
    </GamesProvider>
  );
};
export default App;
