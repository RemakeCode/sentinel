import './empty-state.scss';
import type { FC, ReactNode } from 'react';
import { FileX } from 'lucide-react';

interface EmptyStateProps {
  message?: string;
  description?: string;
  icon?: ReactNode;
  action?: ReactNode;
}

const EmptyState: FC<EmptyStateProps> = ({ message = 'No data available', description, icon, action }) => {
  return (
    <div className='empty-state'>
      <div className='empty-state-icon' aria-hidden='true'>{icon ?? <FileX />}</div>
      <div className='empty-state-content'>
        <h3 className='empty-state-message'>{message}</h3>
        {description && <p className='empty-state-description'>{description}</p>}
      </div>
      {action && <div className='empty-state-action'>{action}</div>}
    </div>
  );
};

export default EmptyState;
