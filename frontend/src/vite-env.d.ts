/// <reference types="vite/client" />

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
