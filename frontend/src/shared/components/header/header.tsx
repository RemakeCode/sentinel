import './header.scss';
import type { FC, HTMLAttributes } from 'react';

export const Header: FC<HTMLAttributes<HTMLElement>> = ({ children, className, ...props }) => {
  return (
    <header {...props} className={`header ${className}`}>
      {children}
    </header>
  );
};
