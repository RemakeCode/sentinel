import type { CSSProperties, FC } from 'react';
import { findClassByName, Focusable, joinClassNames, libraryAssetImageClasses, ProgressBar } from '@decky/ui';
import { IMG_URL } from '@/shared/utils/fetcher';

const Capsule = findClassByName('Capsule')!;
const CapsuleVisible = findClassByName('CapsuleVisible')!;

const styles: Record<string, CSSProperties> = {
  progress: {
    position: 'absolute',
    width: '80%',
    bottom: '10px',
    transform: 'translateX(-50%)',
    zIndex: 2,
    opacity: 0.8
  }
};

export interface LibraryImageProps {
  src: string;
  alt?: string;
  name?: string;
  neverShowTitle?: boolean;
  onError?: () => void;
  progress?: number;
  onActivate?: () => void;
}

const LibraryImage: FC<LibraryImageProps> = ({ src, alt = '', progress, onActivate }) => {
  return (
    <Focusable
      onActivate={onActivate}
      noFocusRing={false}
      className={joinClassNames(
        libraryAssetImageClasses.Container,
        libraryAssetImageClasses.GreyBackground,
        libraryAssetImageClasses.PortraitImage,
        Capsule,
        CapsuleVisible
      )}
    >
      <img
        src={`${IMG_URL}${src}`}
        alt={alt}
        className={joinClassNames(
          libraryAssetImageClasses.Image,
          libraryAssetImageClasses.Visibility,
          libraryAssetImageClasses.Visible
        )}
      />
      {progress !== undefined && (
        <div style={styles.progress}>
          <ProgressBar nProgress={progress} focusable={false} />
        </div>
      )}
    </Focusable>
  );
};

export default LibraryImage;
