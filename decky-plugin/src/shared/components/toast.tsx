import { useLayoutEffect, useRef } from 'react';
import { findModule } from '@decky/ui';
import { ImgIcon } from '@/shared/components/img-icon';
import type { Notification } from '@/shared/types/Notification';
import { acquireToastView } from '@/shared/utils/toast-view';

interface ToastProps {
  message: Notification;
}

export const steamToastClasses = findModule((module) => module?.ShortTemplate && module?.Title && module?.Body) as
  | { Header: string; Title: string; Body: string }
  | undefined;

export const ToastTitle = ({ message }: ToastProps) => {
  return <div className='sentinel-toast-game-name'>{message.GameName}</div>;
};

export const ToastBody = ({ message }: ToastProps) => {
  const rowRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => acquireToastView(rowRef.current), []);

  const progressPercent =
    message.MaxProgress > 0 ? Math.min(100, Math.max(0, (message.Progress / message.MaxProgress) * 100)) : 0;

  return (
    <div ref={rowRef} className='sentinel-toast-layout'>
      <div
        className={
          message.IsRare && !message.IsProgress
            ? 'sentinel-toast-icon sentinel-rare-achievement-glow sentinel-rare-achievement-glow--toast'
            : 'sentinel-toast-icon'
        }
      >
        <ImgIcon src={message.IconPath} />
      </div>
      <div className={`sentinel-toast-details${message.IsProgress ? ' sentinel-toast-details--progress' : ''}`}>
        <div className={`sentinel-toast-title ${steamToastClasses?.Title ?? ''}`}>{message.Title}</div>
        <div className={`sentinel-toast-body ${steamToastClasses?.Body ?? ''}`}>{message.Message}</div>
        {message.IsProgress && (
          <div className='sentinel-toaster-progress-container'>
            <div className='sentinel-toaster-progress-meta'>
              {message.Progress}/{message.MaxProgress}
            </div>
            <progress value={progressPercent} max={100} />
          </div>
        )}
      </div>
    </div>
  );
};
