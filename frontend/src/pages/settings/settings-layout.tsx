import './settings.scss';
import type { FC } from 'react';
import { useState } from 'react';
import { ArrowLeft, Ellipsis, Info, SlidersHorizontal, Trophy } from 'lucide-react';
import { Link, NavLink, Outlet } from 'react-router';

import { GetAppInfo } from '@wa/sentinel/backend/config/file';
import type { AppInfo } from '@wa/sentinel/backend/config/models';

import { HeaderPortal } from '@/shared/components/header/header';

import AboutDialog from './about-dialog';

const SettingsLayout: FC = () => {
  const [aboutDialogOpen, setAboutDialogOpen] = useState(false);
  const [appInfo, setAppInfo] = useState<AppInfo | null>(null);

  const handleAboutDialog = async () => {
    try {
      setAppInfo(await GetAppInfo());
      setAboutDialogOpen(true);
    } catch (err) {
      console.error('Failed to load app info:', err);
    }
  };

  return (
    <div className='full-layout settings-layout'>
      <HeaderPortal>
        <div className='header-nav'>
          <Link to='/'>
            <ArrowLeft />
          </Link>
          <h2>Settings</h2>
        </div>
        <button className='settings-header-about-icon' onClick={handleAboutDialog} title='About' aria-label='About'>
          <Info size={20} />
        </button>
      </HeaderPortal>

      <div data-sidebar-layout='always' className='settings-workspace'>
        <aside data-sidebar aria-label='Settings sections'>
          <nav>
            <ul>
              <li>
                <NavLink to='general'>
                  <SlidersHorizontal size={18} /> Settings
                </NavLink>
              </li>
              <li>
                <NavLink to='achievement-setup'>
                  <Trophy size={18} /> Achievement Setup
                </NavLink>
              </li>
              <li>
                <NavLink to='others'>
                  <Ellipsis size={18} /> Others
                </NavLink>
              </li>
            </ul>
          </nav>
        </aside>
        <main aria-label='Settings content' className='full-layout'>
          <Outlet />
        </main>
      </div>

      <AboutDialog isOpen={aboutDialogOpen} appInfo={appInfo} onClose={() => setAboutDialogOpen(false)} />
    </div>
  );
};

export default SettingsLayout;
