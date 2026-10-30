import { ASSET_URL } from '@/shared/utils/fetcher';
import type { FC, ImgHTMLAttributes } from 'react';

//language=css
const styles = `
  .sentinel-img-icon {
    border: 1px solid #3d4450;
    border-radius: 4px;
    position: relative;
    width: 100%;
    height: 100%;
    box-sizing: border-box;
    object-fit: contain;
  }
`;

export const ImgIcon: FC<ImgHTMLAttributes<HTMLImageElement>> = ({ src, className, ...props }) => {
  return (
    <>
      <style>{styles}</style>
      <img
        className={`sentinel-img-icon${className ? ` ${className}` : ''}`}
        src={`${ASSET_URL}${src}`}
        alt=''
        data-name='ach'
        {...props}
      />
    </>
  );
};
