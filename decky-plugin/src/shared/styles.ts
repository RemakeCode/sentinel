import type { CSSProperties } from 'react';

const styles = {
  wrapper: {
    marginBlock: 'calc(var(--basicui-header-height) + 24px) calc(var(--gamepadui-current-footer-height) + 16px)',
    marginInline: '24px'
  }
} as const satisfies Record<string, CSSProperties>;

export { styles };
