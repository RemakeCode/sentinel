import type { achievement } from '@wa/sentinel/backend/steam/models';

/**
 * Computes the progress percentage based on earned achievements
 * @returns Progress percentage rounded to two decimal places
 * @param achievements
 */
export const computeProgress = (achievements: achievement[] | undefined): number => {
  if (!achievements?.length) return 0;

  if (achievements.length === 0) return 0;

  const earnedCount = achievements.filter((ach) => ach?.CurrentAch.earned).length;

  return Math.round((earnedCount / achievements.length) * 100);
};

const MINUTE = 60;
const HOUR = MINUTE * 60;
const DAY = HOUR * 24;
const WEEK = DAY * 7;
const MONTH = DAY * 30;
const YEAR = DAY * 365;

export const formatRelativeTime = (timestamp: number | undefined): string => {
  if (!timestamp) return '';

  const nowSeconds = Math.floor(Date.now() / 1000);
  const tsSeconds = timestamp > nowSeconds ? Math.floor(timestamp / 1000) : timestamp;
  const diff = nowSeconds - tsSeconds;

  const dateStr = new Date(tsSeconds * 1000).toLocaleDateString();

  if (diff < 0) return dateStr;
  if (diff < MINUTE) return `Just Now (${dateStr})`;
  if (diff < HOUR) return `${Math.round(diff / MINUTE)} minutes ago (${dateStr})`;
  if (diff < DAY) return `${Math.round(diff / HOUR)} hours ago (${dateStr})`;
  if (diff < WEEK) return `${Math.round(diff / DAY)} days ago (${dateStr})`;
  if (diff < MONTH) return `${Math.round(diff / WEEK)} weeks ago (${dateStr})`;
  if (diff < YEAR) return `${Math.round(diff / MONTH)} months ago (${dateStr})`;
  return `${Math.round(diff / YEAR)} years ago (${dateStr})`;
};
