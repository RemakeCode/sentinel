import '@/pages/settings/settings.scss';
import type { FC } from 'react';
import { motion } from 'framer-motion';
import { settingsContentVariants } from '@/pages/settings/settings-motion';
import Logging from '@/pages/settings/others/logging';
import BackupRestore from '@/pages/settings/others/backup-restore';

const Others: FC = () => (
  <motion.section className='settings-pane page-content' variants={settingsContentVariants} initial='hidden' animate='visible'>
    <BackupRestore />
    <Logging />
  </motion.section>
);

export default Others;
