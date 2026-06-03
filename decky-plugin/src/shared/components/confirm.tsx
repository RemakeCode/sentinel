import type { ReactNode } from 'react';
import { ConfirmModal, showModal } from '@decky/ui';

export interface ConfirmModalOptions {
  title: ReactNode;
  description: ReactNode;
  okText?: ReactNode;
  cancelText?: ReactNode;
  destructive?: boolean;
}

export function showConfirmModal(options: ConfirmModalOptions): Promise<boolean> {
  return new Promise((resolve) => {
    const result = showModal(
      <ConfirmModal
        strTitle={options.title}
        strDescription={options.description}
        strOKButtonText={options.okText ?? 'OK'}
        strCancelButtonText={options.cancelText ?? 'Cancel'}
        bDestructiveWarning={options.destructive}
        onOK={() => {
          result.Close();
          resolve(true);
        }}
        onCancel={() => {
          result.Close();
          resolve(false);
        }}
        onEscKeypress={() => {
          result.Close();
          resolve(false);
        }}
      />
    );
  });
}
