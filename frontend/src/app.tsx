import { Outlet, ScrollRestoration } from 'react-router';

const App = () => {
  return (
    <>
      <Outlet />
      <ScrollRestoration />
    </>
  );
};
export default App;
