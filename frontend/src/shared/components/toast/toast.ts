import './toast.scss';

export const showToast = (message: string, variant: 'success' | 'danger' = 'success'): void => {
  window.ot.toast.clear('top-right');
  window.ot.toast(message, undefined, {
    variant: variant === 'success' ? 'warning' : 'danger',
    placement: 'top-right',
    duration: variant === 'danger' ? 8000 : 4000
  });
};
