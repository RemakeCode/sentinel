import './header.scss';
import { FC, HTMLAttributes } from 'react';

export const Header: FC<HTMLAttributes<HTMLElement>> = ({ children, ...props }) => {
  return <header {...props}>{children}</header>;
};
