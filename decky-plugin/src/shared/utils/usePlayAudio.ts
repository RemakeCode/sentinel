import { useCallback, useEffect, useRef } from 'react';
import { ASSET_URL } from '@/shared/utils/fetcher';

let currentAudio: HTMLAudioElement | null = null;

export const playAudio = async (filename: string) => {
  currentAudio?.pause();
  currentAudio = new Audio(`${ASSET_URL}/api/media/media/${filename}`);
  currentAudio.volume = 1;
  await currentAudio.play();
};

export const usePlayAudio = () => {
  const audioRef = useRef<HTMLAudioElement | null>(null);

  const play = useCallback(async (filename: string) => {
    audioRef.current?.pause();
    audioRef.current = new Audio(`${ASSET_URL}/api/media/media/${filename}`);
    audioRef.current.volume = 1;
    await audioRef.current.play();
  }, []);

  useEffect(() => {
    return () => {
      audioRef.current?.pause();
      audioRef.current = null;
    };
  }, []);

  return { play };
};
