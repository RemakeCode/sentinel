export const rareAchievementGlowStyles = `
  @property --rare-achievement-angle {
    syntax: '<angle>';
    inherits: false;
    initial-value: 0deg;
  }

  @keyframes rare-achievement-spin {
    to {
      --rare-achievement-angle: 360deg;
    }
  }

  .sentinel-rare-achievement-glow {
    --rare-achievement-glow-radius: 10px;
    position: relative;
    isolation: isolate;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    overflow: visible;
    border-radius: var(--rare-achievement-glow-radius);
  }

  .sentinel-rare-achievement-glow::before,
  .sentinel-rare-achievement-glow::after {
    content: '';
    position: absolute;
    pointer-events: none;
    border-radius: var(--rare-achievement-glow-radius);
  }

  .sentinel-rare-achievement-glow::before {
    inset: -4px;
    z-index: 0;
    background: conic-gradient(
      from var(--rare-achievement-angle),
      transparent 0deg,
      rgba(255, 214, 77, 0.2) 55deg,
      rgba(255, 174, 0, 0.95) 115deg,
      rgba(255, 244, 178, 0.95) 160deg,
      rgba(255, 174, 0, 0.25) 220deg,
      transparent 300deg
    );
    filter: blur(3px);
    clip-path: inset(0 round var(--rare-achievement-glow-radius));
    animation: rare-achievement-spin 3s linear infinite;
  }

  .sentinel-rare-achievement-glow::after {
    inset: -2px;
    z-index: 1;
    border: 1px solid rgba(255, 214, 77, 0.9);
    box-shadow: 0 0 10px rgba(255, 174, 0, 0.7);
  }

  .sentinel-rare-achievement-glow > img {
    position: relative;
    z-index: 2;
  }

  @media (prefers-reduced-motion: reduce) {
    .sentinel-rare-achievement-glow::before {
      animation: none;
    }
  }
`;
