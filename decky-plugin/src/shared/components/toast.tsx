import type { Notification } from '@/shared/types/Notification';

interface ToastProps {
  message: Notification;
}

export const ToastTitle = ({ message }: ToastProps) => {
  return <div className='sentinel-toast-title'>{message.Title}</div>;
};

export const ToastBody = ({ message }: ToastProps) => {
  return (
    <>
      {!message.IsProgress && <div className='sentinel-toast-body'>{message.Message}</div>}
      {message.IsProgress && (
        <div className='sentinel-toaster-progress-container'>
          <progress value={(message.Progress / message.MaxProgress) * 100} max={100} />
          <div className='sentinel-toaster-progress-meta'>
            {message.Progress}/{message.MaxProgress}
          </div>
        </div>
      )}
    </>
  );
};
