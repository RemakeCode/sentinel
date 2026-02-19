import React from 'react';
import { FileX } from 'lucide-react';
import './EmptyState.scss';

interface EmptyStateProps {
  message?: string;
}

const EmptyState: React.FC<EmptyStateProps> = ({ message = 'No data available' }) => {
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
