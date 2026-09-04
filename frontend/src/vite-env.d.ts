/// <reference types="vite/client" />

declare global {
  namespace React {
    namespace JSX {
      interface IntrinsicElements {
        'ot-tabs': React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement>;
      }
    }
  }

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
