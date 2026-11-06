import '@/pages/settings/settings.scss';
import type { FC } from 'react';
import { motion } from 'framer-motion';
import { settingsContentVariants } from '@/pages/settings/settings-motion';
import BackupRestore from '@/pages/settings/backup-restore/backup-restore';

const BackupRestorePage: FC = () => (
  <motion.section
    className='settings-pane page-content'
    variants={settingsContentVariants}
    initial='hidden'
    animate='visible'
  >
    <header className='settings-page-header'>
      <h2>Backup &amp; Restore</h2>
      <p>Save a copy of your cached game data or restore a previous backup.</p>
    </header>
    <BackupRestore />
  </motion.section>
);

export default BackupRestorePage;
