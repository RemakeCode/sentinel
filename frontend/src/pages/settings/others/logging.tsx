import '@/pages/settings/settings.scss';
import { type FC, useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { Terminal } from 'lucide-react';
import { LoadConfig, SetLoggingEnabled } from '@wa/sentinel/backend/config/file';
import { settingsSectionVariants } from '@/pages/settings/settings-motion';

const Logging: FC = () => {
  const [loggingEnabled, setLoggingEnabled] = useState(false);

  useEffect(() => {
    LoadConfig()
      .then((config) => setLoggingEnabled(config?.logLevel === 'info'))
      .catch(() => window.ot?.toast('Failed to load settings', 'Error', { variant: 'danger' }));
  }, []);

  const handleLoggingToggle = async () => {
    const newValue = !loggingEnabled;
    try {
      await SetLoggingEnabled(newValue);
      setLoggingEnabled(newValue);
      window.ot?.toast(`Logging ${newValue ? 'enabled' : 'disabled'}`, 'Success', { variant: 'success' });
    } catch (err) {
      window.ot?.toast('Failed to update logging setting', 'Error', { variant: 'danger' });
    }
  };

  return (
    <motion.div className='card settings-section' variants={settingsSectionVariants}>
      <h4 className='settings-section-title'>
        <Terminal /> Logging
      </h4>
      <hr className='divider' />
      <div className='settings-grid'>
        <div className='settings-grid-item'>
          <span className='badge' data-variant='success'>
            Console
          </span>
          <span>Enable logging</span>
          <div className='settings-grid-actions'>
            <label className='switch' title='Toggle backend logging'>
              <input type='checkbox' role='switch' checked={loggingEnabled} onChange={handleLoggingToggle} />
            </label>
          </div>
        </div>
      </div>
    </motion.div>
  );
};

export default Logging;
