import '@/pages/settings/settings.scss';
import { showToast } from '@/shared/components/toast';
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
      .catch(() => showToast('Failed to load settings', 'danger'));
  }, []);

  const handleLoggingToggle = async () => {
    const newValue = !loggingEnabled;
    try {
      await SetLoggingEnabled(newValue);
      setLoggingEnabled(newValue);
      showToast(`Logging ${newValue ? 'enabled' : 'disabled'}`);
    } catch (err) {
      showToast('Failed to update logging setting', 'danger');
    }
  };

  return (
    <motion.div className='card settings-section' variants={settingsSectionVariants}>
      <header className='settings-section-header'>
        <h3 className='settings-section-title'>
          Logging
        </h3>
      </header>
      <div className='settings-grid'>
        <div className='settings-grid-item'>
          <div className='settings-grid-details'>
            <strong>Enable logging</strong>
            <p>Print diagnostic information to help troubleshoot issues.</p>
          </div>
          <div className='settings-grid-actions'>
            <label className='switch' title='Toggle backend logging'>
              <input type='checkbox' role='switch' checked={loggingEnabled} onChange={handleLoggingToggle} />
            </label>
            <small>{loggingEnabled ? 'On' : 'Off'}</small>
          </div>
        </div>
      </div>
    </motion.div>
  );
};

export default Logging;
