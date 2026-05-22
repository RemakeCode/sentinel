export type Notification = {
  Title: string;
  Message: string;
  IconPath: string;
  SoundFile: string;
  GameName: string;
  Progress: number;
  MaxProgress: number;
  IsProgress: boolean; // true, skip Message
};

