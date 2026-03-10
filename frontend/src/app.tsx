import '@wailsio/runtime';

import { Outlet, ScrollRestoration } from 'react-router';
import { Events } from '@wailsio/runtime';

Events.On('sentinel::ready', () => {
  console.log(`[Event Registered]`);
});

const App = () => {
  return (
    <>
      <Outlet />
      <ScrollRestoration />
    </>
  );
};
export default App;
