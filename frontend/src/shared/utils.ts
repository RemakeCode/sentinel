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
