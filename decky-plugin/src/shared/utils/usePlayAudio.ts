import { useCallback, useEffect, useRef } from 'react';
import { BASE_URL } from '@/shared/utils/fetcher';

export const usePlayAudio = () => {
  const audioRef = useRef<HTMLAudioElement | null>(null);

  const play = useCallback((filename: string) => {
    audioRef.current?.pause();
    audioRef.current = new Audio(`${BASE_URL}/sentinel-assets/media/${filename}`);
    audioRef.current.play();
  }, []);

  useEffect(() => {
    return () => {
      audioRef.current?.pause();
      audioRef.current = null;
    };
  }, []);

  return { play };
};
