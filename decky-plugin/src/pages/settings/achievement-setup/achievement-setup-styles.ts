export const achievementSetupStyles = `
  .sentinel-gbe-setup-content {
    box-sizing: border-box;
    height: 160px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .sentinel-gbe-setup-qr {
    display: grid;
    grid-template-columns: 160px minmax(0, 1fr);
    align-items: center;
    gap: 16px;
    min-height: 160px;
  }

  .sentinel-gbe-setup-qr-code {
    width: 160px;
    height: 160px;
    overflow: hidden;
    background: white;
    border-radius: 6px;
  }

  .sentinel-gbe-setup-qr-image,
  .sentinel-gbe-setup-qr-image svg {
    width: 100%;
    height: 100%;
    display: block;
  }

  .sentinel-gbe-setup-qr-image {
    position: relative;
  }

  .sentinel-gbe-setup-qr-overlay {
    position: absolute;
    top: 50%;
    left: 50%;
    box-sizing: border-box;
    display: block;
    width: 32px;
    height: 32px;
    padding: 3px;
    border-radius: 50%;
    background: white;
    transform: translate(-50%, -50%);
  }

  .sentinel-gbe-setup-qr-image--blurred {
    filter: blur(6px);
    transform: scale(1.04);
  }

  .sentinel-gbe-setup-qr-message {
    overflow-wrap: anywhere;
    text-align: center;
  }

  .sentinel-gbe-setup-error {
    color: var(--gpColor-Red, #f04747);
  }

  .sentinel-gbe-setup-qr-details {
    align-self: stretch;
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 12px;
  }

  .sentinel-gbe-setup-title {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    width: 100%;
    padding-bottom: 8px;
    border-bottom: 1px solid rgba(100, 100, 100, 0.4);
  }

  .sentinel-gbe-setup-title-text {
    flex: 1;
    min-width: 0;
  }

  .sentinel-gbe-setup-progress {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 4px;
    width: min(46%, 240px);
    flex-shrink: 0;
    text-align: right;
    color: var(--gpColor-Blue, #1a9fff);
    font-size: 12px;
    line-height: 12px;
    text-transform: uppercase;
  }

  .sentinel-gbe-setup-progress > :last-child {
    width: 100%;
  }

  .sentinel-gbe-setup-progress progress {
    width: 100%;
    accent-color: var(--gpColor-Blue, #1a9fff);
    font-weight: 700;
  }

  .sentinel-gbe-setup-undo-content {
    box-sizing: border-box;
    min-height: 0;
    display: flex;
    align-items: flex-start;
  }

  @media (max-width: 600px) {
    .sentinel-gbe-setup-title {
      flex-direction: column;
    }

    .sentinel-gbe-setup-progress {
      width: 100%;
      align-items: stretch;
      text-align: left;
    }
  }
`;
