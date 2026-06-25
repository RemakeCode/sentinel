import './library-sync-alert.scss';
import type { FC } from 'react';
import type { LibrarySyncStatus } from '@wa/sentinel/backend/steam';
import { AnimatePresence, motion } from 'framer-motion';

interface LibrarySyncAlertProps {
  syncStatus: LibrarySyncStatus;
}

const alertVariants = {
  initial: { opacity: 0, y: -12 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -12 }
};

const LibrarySyncAlert: FC<LibrarySyncAlertProps> = ({ syncStatus }) => {
  const isRunning = syncStatus.State === 'running';

  return (
    <div className='library-sync-alert' aria-live='polite'>
      <AnimatePresence>
        {isRunning && (
          <motion.div
            className='alert info'
            role='alert'
            aria-busy='true'
            data-spinner='small'
            variants={alertVariants}
            initial='initial'
            animate='animate'
            exit='exit'
            transition={{ duration: 0.2, ease: 'easeInOut' }}
          >
            <span className='library-sync-alert-message'>Fetching metadata</span>
            <span className='library-sync-alert-count'>
              {syncStatus.Current}/{syncStatus.Total}
            </span>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default LibrarySyncAlert;
