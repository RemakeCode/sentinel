import { FC } from 'react';
import { Focusable, joinClassNames, libraryAssetImageClasses, ProgressBar } from '@decky/ui';
import { ASSET_URL } from '@/shared/utils/fetcher';

//language=css
const libraryImageStyles = `
  .sentinel-library-image-wrapper {
    width: inherit;
  }
  .sentinel-library-image-progress {
    position: absolute;
    width: 80%;
    bottom: 10px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 2;
    opacity: 0.8;
  }
`;

export interface LibraryImageProps {
  src: string;
  alt?: string;
  name?: string;
  onError?: () => void;
  progress?: number;
  onActivate?: () => void;
}

const LibraryImage: FC<LibraryImageProps> = ({ src, alt = '', progress, onActivate }) => {
  return (
    <>
      <style>{libraryImageStyles}</style>
      <Focusable
        onActivate={onActivate}
        noFocusRing={false}
        className={joinClassNames(
          libraryAssetImageClasses.Container,
          libraryAssetImageClasses.GreyBackground,
          libraryAssetImageClasses.PortraitImage,
          'sentinel-library-image-wrapper'
        )}
      >
        <img
          src={`${ASSET_URL}${src}`}
          alt={alt}
          className={joinClassNames(
            libraryAssetImageClasses.Image,
            libraryAssetImageClasses.Visibility,
            libraryAssetImageClasses.Visible
          )}
        />
        {progress !== undefined && (
          <div className='sentinel-library-image-progress'>
            <ProgressBar nProgress={progress} focusable={false} />
          </div>
        )}
      </Focusable>
    </>
  );
};

export { LibraryImage };
