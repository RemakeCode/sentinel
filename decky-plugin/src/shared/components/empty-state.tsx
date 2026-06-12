import type { FC } from 'react';
import { DialogButton, DialogLabel } from '@decky/ui';
import { PiGameController } from 'react-icons/pi';

export type EmptyStateVariant = 'library' | 'main';

export interface EmptyStateProps {
  label: string;
  buttonText: string;
  buttonClick: () => void;
  variant: EmptyStateVariant;
}

//language=css
const emptyStateStyles = `
  .sentinel-empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }

  .sentinel-empty-state--main {
    margin-block-start: 64px;
  }

  .sentinel-empty-state--library {
    height: 100%;
    justify-content: center;
  }

  .sentinel-empty-state-label {
    font-weight: 700;
  }

  .sentinel-empty-state-icon {
    fill: #acb2b8;
    animation: sentinel-empty-state-wobble 1s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .sentinel-empty-state-button {
    margin-block-start: 24px;
    width: auto;
  }

  .sentinel-empty-state--library .sentinel-empty-state-label {
    font-size: 32px;
  }

  .sentinel-empty-state--library .sentinel-empty-state-icon {
    width: 80px;
    height: 80px;
  }

  .sentinel-empty-state--main .sentinel-empty-state-label {
    font-size: 16px;
  }

  .sentinel-empty-state--main .sentinel-empty-state-icon {
    width: 60px;
    height: 60px;
  }

  .sentinel-empty-state--main .sentinel-empty-state-button {
    margin-block-start: 8px;
    width: auto;
  }

  @keyframes sentinel-empty-state-wobble {
    0%, 100% {
      transform: rotate(0deg);
    }
    25% {
      transform: rotate(15deg);
    }
    50% {
      transform: rotate(0deg);
    }
    75% {
      transform: rotate(-15deg);
    }
  }
`;

const EmptyState: FC<EmptyStateProps> = ({ label, buttonText, buttonClick, variant }) => (
  <>
    <style>{emptyStateStyles}</style>
    <div className={`sentinel-empty-state sentinel-empty-state--${variant}`}>
      <DialogLabel className='sentinel-empty-state-label'>{label}</DialogLabel>
      <PiGameController className='sentinel-empty-state-icon' />
      <DialogButton className='sentinel-empty-state-button' onClick={buttonClick}>
        {buttonText}
      </DialogButton>
    </div>
  </>
);

export { EmptyState };
