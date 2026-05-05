import { IMG_URL } from '@/shared/utils/fetcher';
import { CSSProperties } from 'react';

const style: CSSProperties = {
  width: '60px',
  border: '1px solid #3d4450',
  borderRadius: '4px'
};
export const ImgIcon = ({ src }: { src: string }) => {
  return <img style={style} src={`${IMG_URL}${src}`} alt='' data-name='ach' />;
};
