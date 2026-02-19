import React, { useState } from 'react';
import './dashboard.scss';
import { Search, Bell, Settings, Home } from 'lucide-react';
import { Link } from 'react-router';
import EmptyState from '@/shared/components/EmptyState';

const Dashboard: React.FC = () => {
  const [games, setGames] = useState<[]>([]);
  const [loading, setLoading] = useState(false);
  return (
    <div className='dashboard-layout'>
      <div className='dashboard-main-content'>
        <div>
          {/* Header */}
          <div className='dashboard-header'>
            <div className='dashboard-header-left'>
              <button className='dashboard-header-left-home-btn'>
                <Home />
              </button>
              <div className='dashboard-header-left-search-bar'>
                <Search className='dashboard-header-left-search-bar-search-icon' />
                <input type='text' placeholder='Tango...' />
              </div>
            </div>
            <div className='dashboard-header-tools'>
              <button className='dashboard-header-tools-icon-btn'>
                <Bell />
              </button>
              <Link to='/settings' viewTransition className='dashboard-settings-link'>
                <div className='dashboard-header-tools-settings-entry-point'>
                  <button className='dashboard-header-tools-icon-btn'>
                    <Settings className='dashboard-header-tools-settings-entry-point-icon' />
                  </button>
                </div>
              </Link>
            </div>
          </div>

          {/* Most Played Games */}
          <div className='dashboard-section-container'>
            <div className='dashboard-section-title'>
              <strong className='dashboard-section-title-label'>
                Library
              </strong>
              <span className='dashboard-section-title-action'>View All</span>
            </div>

            {loading ? (
              <div className='dashboard-loading-container'>
                <div className='spinner' data-size='large'></div>
              </div>
            ) : games.length === 0 ? (
              <EmptyState message='No games found. Add an emulator path in settings!' />
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
