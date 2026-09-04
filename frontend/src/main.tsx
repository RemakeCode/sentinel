import '@knadh/oat/oat.min.js';
import '@/shared/styles/global.scss';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { createHashRouter, Navigate, RouterProvider } from 'react-router';
import GameDetails from '@/pages/game-details/game-details';
import Settings from '@/pages/settings/settings';
import SettingsLayout from '@/pages/settings/settings-layout';
import Others from '@/pages/settings/others';
import AchievementSetup from '@/pages/settings/achievement-setup/achievement-setup';
import Dashboard from '@/pages/dashboard/dashboard';
import App from '@/app';

const container = document.getElementById('root');
const root = createRoot(container!);

const router = createHashRouter([
  {
    path: '/',
    element: <App />,
    children: [
      {
        index: true,
        element: <Dashboard />
      },
      {
        path: '/game/:id',
        element: <GameDetails />
      },
      {
        path: '/settings',
        element: <SettingsLayout />,
        children: [
          {
            index: true,
            element: <Navigate to='general' replace />
          },
          {
            path: 'achievement-setup',
            element: <AchievementSetup />
          },
          {
            path: 'general',
            element: <Settings />
          },
          {
            path: 'others',
            element: <Others />
          }
        ]
      }
    ]
  }
]);

root.render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>
);
