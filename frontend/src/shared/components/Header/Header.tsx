import './Header.scss';
import { DetailedHTMLProps, FC, HTMLAttributes, PropsWithChildren } from 'react';

export const Header: FC<HTMLAttributes<HTMLElement>> = ({ children, ...props }) => {
  return <header {...props}>{children}</header>;
};
