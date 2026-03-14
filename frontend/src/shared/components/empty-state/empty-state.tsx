import type { FC } from 'react';
import { FileX } from 'lucide-react';
import './empty-state.scss';

interface EmptyStateProps {
  message?: string;
}

const EmptyState: FC<EmptyStateProps> = ({ message = 'No data available' }) => {
  return (
    <div className='empty-state'>
      <div className='empty-state-content'>
        <FileX className='empty-state-icon' />
        <p className='empty-state-message'>{message}</p>
      </div>
    </div>
  );
};

export default EmptyState;
