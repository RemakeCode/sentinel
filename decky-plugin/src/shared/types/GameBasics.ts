export interface Achievement {
  earned: boolean;
  earned_time: number;
  max_progress?: number;
  progress?: number;
}

export interface AchievementInfo {
  Name: string;
  DisplayName: string;
  Description: string;
  Icon: string;
  IconGray: string;
  DefaultValue: number;
  Hidden: number;
  CurrentAch?: Achievement;
}

export interface GameBasics {
  AppID: string;
  Name: string;
  HeaderImage: string;
  PortraitImage: string;
  Achievement: {
    Total: number;
    List: AchievementInfo[];
  };
}
