import type { FC } from 'react';
import { motion } from 'framer-motion';
import { EyeOff } from 'lucide-react';
import type { GlobalAchievementPercentage, achievement } from '@wa/sentinel/backend/steam/models';

const itemVariants = {
  hidden: { opacity: 0, y: 15 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.3,
      ease: 'easeOut'
    }
  }
};

export type AchievementListItemProps = {
  ach: achievement;
  globalPercentages: Map<string, GlobalAchievementPercentage>;
};

export const formatUnlockTime = (timestamp: number | undefined): string => {
  if (!timestamp) return '';
  const tsSeconds = timestamp > Math.floor(Date.now() / 1000) ? Math.floor(timestamp / 1000) : timestamp;
  return new Date(tsSeconds * 1000).toLocaleString();
};

export const AchievementListItem: FC<AchievementListItemProps> = ({ ach, globalPercentages }) => {
  const currentAch = (ach as any).CurrentAch;
  const earned = currentAch?.earned;
  const hasProgress = (currentAch?.max_progress || 0) > 1;
  const progress = currentAch?.progress || 0;
  const maxProgress = currentAch?.max_progress || 1;
  const displayProgress = earned && progress !== maxProgress ? progress + 1 : progress;
  const globalPercentage = globalPercentages.get(ach.Name);
  const isRare = Boolean(earned && globalPercentage?.isRare);

  return (
    <motion.li className='game-details-ach-item' variants={itemVariants}>
      <div className={`game-details-ach-icon${isRare ? ' game-details-ach-icon--rare' : ''}`}>
        <img src={ach.Icon} alt={ach.DisplayName} width={64} height={64} />
      </div>
      <div className='game-details-ach-info'>
        <span className='game-details-ach-title'>{ach.DisplayName}</span>
        <span className='game-details-ach-desc'>
          <span className={`${ach.Hidden === 1 ? 'blur' : ''}`}>{ach.Description || ''}</span>
          {ach.Hidden === 1 && <EyeOff width={18} height={18} />}
        </span>
        {hasProgress && (
          <div className='game-details-ach-progress'>
            <progress value={displayProgress} max={maxProgress} />
            <span className='game-details-ach-progress-text'>
              {displayProgress} / {maxProgress}
            </span>
          </div>
        )}
      </div>
      <div className='game-details-ach-meta'>
        <code className='game-details-ach-unlocktime'>
          {currentAch?.earned_time ? formatUnlockTime(currentAch.earned_time) : 'Locked'}
        </code>
        {globalPercentage && (
          <code className='game-details-ach-global-percent fade-in'>
            {globalPercentage.percent}% of players have this
          </code>
        )}
      </div>
    </motion.li>
  );
};
