import './app.scss';
import type { FC } from 'react';
import { useEffect, useRef, useState } from 'react';
import { AnimatePresence, motion, type Variants } from 'framer-motion';
import { ScrollRestoration, useLocation, useOutlet } from 'react-router';
import { GetAppInfo } from '@wa/sentinel/backend/config/file';
import { GamesProvider } from '@/shared/context/games-context';
import { Header } from '@/shared/components/header/header';

// Spatial Layered Zoom variants based on navigation direction
const pageVariants: Variants = {
  initial: (direction: 'into' | 'out') => ({
    opacity: 0,
    scale: direction === 'into' ? 0.95 : 1.05,
    y: direction === 'into' ? 20 : -10
  }),
  animate: {
    opacity: 1,
    scale: 1,
    y: 0,
    transition: {
      duration: 0.3,
      ease: [0.22, 1, 0.36, 1] // Sharp exit, smooth entry (easeOutExpo-ish)
    }
  },
  exit: (direction: 'into' | 'out') => ({
    opacity: 0,
    scale: direction === 'into' ? 0.98 : 1.05,
    y: direction === 'into' ? -10 : 20,
    transition: {
      duration: 0.2,
      ease: 'easeIn'
    }
  })
};

const App: FC = () => {
  const [ready, setReady] = useState(false);
  const location = useLocation();
  const outlet = useOutlet();

  // Track depth to determine animation direction
  const prevDepth = useRef(0);
  const getDepth = (path: string) => {
    if (path === '/') return 0;
    if (path.startsWith('/game/')) return 1;
    if (path === '/settings') return 1;
    return 0;
  };

  const currentDepth = getDepth(location.pathname);
  const direction = currentDepth > prevDepth.current ? 'into' : 'out';

  useEffect(() => {
    prevDepth.current = currentDepth;
  }, [currentDepth]);

  useEffect(() => {
    GetAppInfo()
      .then(() => setReady(true))
      .catch(() => setReady(true));
  }, []);

  if (!ready) {
    return (
      <div className='app-loader'>
        <div className='hstack'>
          <div aria-busy='true' data-spinner='large' />
          <span>Loading Sentinel ...</span>
        </div>
      </div>
    );
  }

  return (
    <GamesProvider>
      <Header id='global-header'>
        <div id='header-portal-root' className='header-portal-root' />
      </Header>
      <AnimatePresence mode='wait' custom={direction}>
        <motion.div
          key={location.pathname}
          custom={direction}
          initial='initial'
          animate='animate'
          exit='exit'
          variants={pageVariants}
          className='page-transition-wrapper'
        >
          {outlet}
        </motion.div>
      </AnimatePresence>
      <ScrollRestoration />
    </GamesProvider>
  );
};

export default App;
