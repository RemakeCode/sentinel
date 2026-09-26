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
  const hasFailures = syncStatus.State === 'error';
  const message = hasFailures
    ? syncStatus.Failed > 0
      ? 'Metadata sync completed with failures'
      : 'Metadata sync failed'
    : 'Fetching metadata';
  const count =
    hasFailures && syncStatus.Failed > 0
      ? `${syncStatus.Failed}/${syncStatus.Total} failed`
      : `${syncStatus.Current}/${syncStatus.Total}`;

  return (
    <div className='library-sync-alert' aria-live='polite'>
      <AnimatePresence>
        {(isRunning || hasFailures) && (
          <motion.div
            className='alert'
            role='alert'
            aria-busy={isRunning}
            data-spinner={isRunning ? 'small' : undefined}
            data-variant={hasFailures ? 'warning' : undefined}
            variants={alertVariants}
            initial='initial'
            animate='animate'
            exit='exit'
            transition={{ duration: 0.2, ease: 'easeInOut' }}
          >
            <span className='library-sync-alert-message'>{message}</span>
            <span className='library-sync-alert-count'>{count}</span>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default LibrarySyncAlert;
