import { ASSET_URL } from '@/shared/utils/fetcher';
import { CSSProperties } from 'react';

const style: CSSProperties = {
  height: '40px',
  border: '1px solid #3d4450',
  borderRadius: '4px',
  position: 'relative',
  bottom: '-2px'
};
export const ImgIcon = ({ src }: { src: string }) => {
  return <img style={style} src={`${ASSET_URL}${src}`} alt='' data-name='ach' />;
};
