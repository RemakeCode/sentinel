import './header.scss';
import type { FC, HTMLAttributes, PropsWithChildren } from 'react';
import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';

export const Header: FC<HTMLAttributes<HTMLElement>> = ({ children, className, ...props }) => {
  return (
    <header {...props} className={`header ${className}`}>
      {children}
    </header>
  );
};

export const HeaderPortal: FC<PropsWithChildren> = ({ children }) => {
  const [target, setTarget] = useState<HTMLElement | null>(() => document.getElementById('header-portal-root'));

  useEffect(() => {
    setTarget(document.getElementById('header-portal-root'));
  }, []);

  if (!target) {
    return null;
  }

  return createPortal(children, target);
};
