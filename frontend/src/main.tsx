import '@wailsio/runtime';
import React, { useLayoutEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { createHashRouter, RouterProvider, ScrollRestoration, useLocation } from 'react-router';
import '@/shared/styles/global.scss';
import '@knadh/oat/oat.min.js';
import GameDetails from '@/pages/game-details/game-details';
import Settings from '@/pages/settings/settings';
import Dashboard from '@/pages/dashboard/dashboard';
import { ScrollToTop } from '@/shared/components/ScrollToTop/scroll-to-top';
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
        element: <Settings />
      }
    ]
  }
]);

root.render(
  <React.StrictMode>
    <RouterProvider router={router} />
  </React.StrictMode>
);
