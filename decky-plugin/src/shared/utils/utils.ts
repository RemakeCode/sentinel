import { Fetcher } from '@/shared/utils/fetcher';
import { getExternalResourceURL } from '@decky/api';
import type { achievement as AchievementInfo } from '@/shared/types/_generated/sentinel/backend/steam/models';

const fetcher = new Fetcher();

interface SteamTab {
  title: string;
}

let notificationTabCache: string | undefined;

export function computeProgress(list: AchievementInfo[] | null): number {
  if (!list || list.length === 0) return 0;
  const earned = list.filter((a) => a.CurrentAch?.earned).length;
  return Math.round((earned / list.length) * 100);
}

//cache notification tab
export const getNotificationTab = async () => {
  if (!notificationTabCache) {
    const response = await fetcher.get<Array<SteamTab>>(getExternalResourceURL('http://localhost:8080/json'));
    const notifyTab = response.find((data: SteamTab) => data.title.toLowerCase().includes('notification'));
    notificationTabCache = notifyTab?.title;
    return notifyTab?.title;
  }
  return notificationTabCache;
};
