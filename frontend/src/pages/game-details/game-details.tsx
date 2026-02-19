import React, { useEffect } from 'react';
import { useParams, Link } from 'react-router';
import { ArrowLeft } from 'lucide-react';
import './game-details.scss';

const GameDetails: React.FC = () => {
    const { id } = useParams<{ id: string }>();

    // Mock data based on ID (deterministic enough for demo)
    const imageSeed = parseInt(id || '1') + 100;
    const imageUrl = `https://picsum.photos/seed/${imageSeed}/300/450`;

    return (
        <div className="game-details-container">
            <div className="game-details-container-image-section">
                <Link to="/" viewTransition className="game-details-back-link">
                    <div className="game-details-container-image-section-back-button">
                        <ArrowLeft className="game-details-back-icon" /> Back to Dashboard
                    </div>
                </Link>
                <img
                    src={imageUrl}
                    alt={`Game ${id}`}
                    className="game-details-container-image-section-large-image"
                    data-view-transition={`game-image-${id}`}
                />
            </div>

            <div className="game-details-container-info-section">
                <h1>Game Title {parseInt(id || '0') + 1}</h1>

                <ul className="game-details-container-info-section-dummy-list">
                    {Array.from({ length: 5 }).map((_, i) => (
                        <li key={i}>
                            <span className="game-details-container-info-section-dummy-list-item-title">Achievement {i + 1}</span>
                            <span className="game-details-container-info-section-dummy-list-item-desc">Unlocked at {new Date().toLocaleDateString()}</span>
                        </li>
                    ))}
                </ul>
            </div>
        </div>
    );
};

export default GameDetails;
