import '@/pages/settings/settings.scss';
import type { FC } from 'react';
import { motion } from 'framer-motion';
import { settingsContentVariants } from '@/pages/settings/settings-motion';
import Logging from '@/pages/settings/others/logging';

const Others: FC = () => (
  <motion.section
    className='settings-pane page-content'
    variants={settingsContentVariants}
    initial='hidden'
    animate='visible'
  >
    <header className='settings-page-header'>
      <h2>Other settings</h2>
      <p>Diagnostic tools and application logging.</p>
    </header>
    <Logging />
  </motion.section>
);

export default Others;
