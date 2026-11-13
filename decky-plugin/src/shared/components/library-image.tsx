import { FC, useEffect, useRef, useState } from 'react';
import { findClassModule, Focusable, joinClassNames, libraryAssetImageClasses, ProgressBar } from '@decky/ui';
import { ASSET_URL } from '@/shared/utils/fetcher';

//language=css
const libraryImageStyles = `
  .sentinel-library-image-wrapper {
    width: inherit;
  }
  .sentinel-library-image-wrapper.sentinel-library-image-wrapper--steam-card {
    height: 0;
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
  .sentinel-library-image-refreshing {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.55);
    z-index: 3;
    font-size: 14px;
    font-weight: 700;
  }
`;

const steamLibraryCardClass = (
  findClassModule((classes) => classes.LibraryItemBox && classes.InCollection) || undefined
)?.LibraryItemBox;

export interface LibraryImageProps {
  src: string;
  fallbackSrc?: string;
  alt?: string;
  name?: string;
  onError?: () => void;
  progress?: number;
  onActivate?: () => void;
  onOpenContextMenu?: (parent?: EventTarget | null) => void;
  isRefreshing?: boolean;
  autoFocus?: boolean;
  onGamepadFocus?: () => void;
  steamCard?: boolean;
}

const LibraryImage: FC<LibraryImageProps> = ({
  src,
  fallbackSrc,
  alt = '',
  onError,
  progress,
  onActivate,
  onOpenContextMenu,
  isRefreshing = false,
  autoFocus,
  onGamepadFocus,
  steamCard = false
}) => {
  const focusableRef = useRef<HTMLDivElement | null>(null);
  const [activeSrc, setActiveSrc] = useState(src);
  const useSteamCard = steamCard && Boolean(steamLibraryCardClass);

  useEffect(() => {
    setActiveSrc(src);
  }, [src]);

  const handleImageError = () => {
    onError?.();
    if (fallbackSrc && activeSrc !== fallbackSrc) {
      setActiveSrc(fallbackSrc);
    }
  };

  return (
    <>
      <style>{libraryImageStyles}</style>
      <Focusable
        ref={focusableRef}
        autoFocus={autoFocus}
        {...{ onGamepadFocus }}
        onActivate={onActivate}
        onMenuButton={() => onOpenContextMenu?.(focusableRef.current)}
        onMenuActionDescription='Game Actions'
        onContextMenu={(event) => {
          event.preventDefault();
          onOpenContextMenu?.(focusableRef.current ?? event.currentTarget);
        }}
        noFocusRing
        className={joinClassNames(
          libraryAssetImageClasses.Container,
          libraryAssetImageClasses.GreyBackground,
          libraryAssetImageClasses.PortraitImage,
          'sentinel-library-image-wrapper',
          useSteamCard ? 'sentinel-library-image-wrapper--steam-card' : '',
          useSteamCard ? steamLibraryCardClass || '' : ''
        )}
      >
        <img
          src={`${ASSET_URL}${activeSrc}`}
          alt={alt}
          onError={handleImageError}
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
        {isRefreshing && <div className='sentinel-library-image-refreshing'>Refreshing...</div>}
      </Focusable>
    </>
  );
};

export { LibraryImage };
