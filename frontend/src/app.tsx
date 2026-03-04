import { Outlet, ScrollRestoration } from 'react-router';
import { ScrollToTop } from '@/shared/components/ScrollToTop/scroll-to-top';

const App = () => {
  return (
    <>
      <Outlet />
      <ScrollRestoration />
    </>
  );
};
export default App;
