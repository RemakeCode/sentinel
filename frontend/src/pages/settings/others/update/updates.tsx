import '@/pages/settings/settings.scss';
import './updates.scss';
import { type FC, useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { GetAppInfo } from '@wa/sentinel/backend/config/file';
import { useUpdate } from '@/shared/context/update-context';
import { settingsSectionVariants } from '@/pages/settings/settings-motion';
import UpdateDialog from '@/pages/settings/others/update/update-dialog';
import ReleaseNotes from '@/pages/settings/others/update/release-notes';

const Updates: FC = () => {
  const [version, setVersion] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const { release, checking, error, check } = useUpdate();

  useEffect(() => {
    GetAppInfo().then((info) => setVersion(info.version))
      .catch(() => setVersion('Unavailable'));
  }, []);

  return (
    <motion.section className='card settings-section' aria-labelledby='updates-title' variants={settingsSectionVariants}>
      <header className='settings-section-header'>
        <h3 id='updates-title' className='settings-section-title'>Updates</h3>
      </header>
      <div className='settings-grid'>
        <div className='settings-grid-item settings-grid-item-preference'>
          <div className='settings-details'>
            <strong className='settings-details-label'>Current version: {version || 'Loading…'}</strong>
            <p className={`settings-details-description${error ? ' updates-error' : ''}`} role='status'>
              {checking ? 'Checking for updates…' : release ? `Version ${release.version} is available.` : error || 'You’re up to date.'}
            </p>
          </div>
          <div className='settings-grid-actions'>
            <button type='button' className='outline' disabled={checking || release !== null} onClick={() => void check()}>Check for Updates</button>
          </div>
        </div>
      </div>
      {release && (
        <div className='settings-section-body'>
          <div className='settings-inset updates-notes'>
            <h4 className='settings-details-label'>What’s new in {release.version}</h4>
            <ReleaseNotes notes={release.notes ?? ''} className='updates-excerpt' />
            <div className='settings-grid-actions updates-notes-actions'>
              <button type='button' onClick={() => setDialogOpen(true)}>Download</button>
            </div>
          </div>
        </div>
      )}
      <UpdateDialog isOpen={dialogOpen} version={release?.version ?? ''} notes={release?.notes ?? ''} onClose={() => setDialogOpen(false)} />
    </motion.section>
  );
};

export default Updates;
