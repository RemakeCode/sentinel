import './settings.scss';
import type { FC } from 'react';
import { useEffect, useState } from 'react';
import { Terminal } from 'lucide-react';

import { LoadConfig, SetLoggingEnabled } from '@wa/sentinel/backend/config/file';

const Others: FC = () => {
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
    <section className='settings-pane page-content'>
      <div className='card settings-section settings-section-first'>
        <h4 className='settings-section-title'>
          <Terminal /> Logging
        </h4>
        <hr className='divider' />
        <div className='settings-grid'>
          <div className='settings-grid-item'>
            <span className='badge success'>Console</span>
            <span>Enable logging</span>
            <label className='switch' title='Toggle backend logging'>
              <input type='checkbox' role='switch' checked={loggingEnabled} onChange={handleLoggingToggle} />
            </label>
            <div />
          </div>
        </div>
      </div>
    </section>
  );
};

export default Others;
