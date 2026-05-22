import { Fetcher } from '@/shared/utils/fetcher';
import { getExternalResourceURL } from '@decky/api';

const fetcher = new Fetcher();

interface SteamTab {
  title: string;
}

let notificationTabCache: string | undefined;
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
