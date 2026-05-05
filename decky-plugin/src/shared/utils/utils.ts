import { Fetcher } from '@/shared/utils/fetcher';

const fetcher = new Fetcher();

interface SteamTab {
  title: string;
}

//cache notification tab
export const getNotificationTab = async () => {
  const response = await fetcher.get<Array<SteamTab>>('http://localhost:8080/json');
  const notifyTab = response.find((data: SteamTab) => data.title.toLowerCase().includes('notification'));
  return notifyTab?.title;
};
