import { ASSET_URL } from '@/shared/utils/fetcher';
import { CSSProperties, FC, ImgHTMLAttributes } from 'react';

const imgStyle: CSSProperties = {
  height: '40px',
  border: '1px solid #3d4450',
  borderRadius: '4px',
  position: 'relative',
  bottom: '-1px'
};

export const ImgIcon: FC<ImgHTMLAttributes<HTMLImageElement>> = ({ src, style, ...props }) => {
  return <img style={{ ...imgStyle, ...style }} src={`${ASSET_URL}${src}`} alt='' data-name='ach' {...props} />;
};
