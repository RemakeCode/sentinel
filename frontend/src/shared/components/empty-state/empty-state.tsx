import type { FC, ReactNode } from 'react';
import { FileX } from 'lucide-react';
import './empty-state.scss';

interface EmptyStateProps {
  message?: string;
  icon: ReactNode;
}

const EmptyState: FC<EmptyStateProps> = ({ message = 'No data available', icon }) => {
  return (
    <div className='empty-state'>
      <div className='empty-state-content'>
        <div className='empty-state-icon'>{icon ? icon : <FileX />}</div>
        <p className='empty-state-message'>{message}</p>
      </div>
    </div>
  );
};

export default EmptyState;
