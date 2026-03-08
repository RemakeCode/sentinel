import React from 'react';
import { Link, useLocation, useParams } from 'react-router';
import { ArrowLeft, EyeOff } from 'lucide-react';
import { GameBasics } from '@wa/sentinel/backend/steam';
import './game-details.scss';
import { Header } from '@/shared/components/Header/Header';

const GameDetails: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const game = location.state?.game as GameBasics | undefined;

  return (
    <main className='full-layout'>
      <Header className={'game-details-header'}>
        <Link to='/' viewTransition>
          <ArrowLeft />
        </Link>
        <h2>Achievements</h2>
      </Header>
      <section className='main-content'>
        <div className='game-details-section'>
          <div className='game-details-container'>
            <div className='game-details-container-inner'>
              <div
                className='game-details-image card'
                style={{ viewTransitionName: `game-image-${id}` } as React.CSSProperties}
              >
                <img src={game?.PortraitImage} alt={game?.Name} />
              </div>

              <div className='game-details-stats'>
                <div>{game?.Name} 2021</div>
                <pre></pre>
              </div>
            </div>
          </div>
          <div className='game-details-ach'>
            <h1>{game?.Name}</h1>
            <ul className='game-details-ach-list'>
              {game?.Achievement.List.map((ach, i) => (
                <li key={i} className='game-details-ach-item'>
                  <div className='game-details-ach-icon'>
                    <img src={ach.Icon} alt={ach.DisplayName} width={64} height={64} />
                  </div>
                  <div className='game-details-ach-info'>
                    <span className='game-details-ach-title'>{ach.DisplayName}</span>
                    <span className='game-details-ach-desc'>
                      <span className={`${ach.Hidden === 1 ? 'blur' : ''}`}>{ach.Description || ''}</span>
                      {ach.Hidden === 1 && <EyeOff width={18} height={18} />}
                    </span>
                  </div>
                  <div className='game-details-ach-unlocktime'></div>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>
    </main>
  );
};

export default GameDetails;
