/// <reference types="vite/client" />

declare namespace JSX {
  interface IntrinsicElements {
    'ot-tabs': React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement>;
  }
}

declare global {
  interface Window {
    ot: {
      toast: (
        message: string,
        title?: string,
        options?: { variant?: 'success' | 'danger' | 'info' | 'warning' }
      ) => void;
    };
  }
}

export {};
