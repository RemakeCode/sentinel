export interface EventSourceClient {
  addEventListener(type: string, listener: (event: MessageEvent<string>) => void): void;
  close(): void;
}

interface SSELogger {
  info(message: string): void;
  warn(message: string): void;
}

interface SSEControllerOptions {
  url: string;
  onMessage: (event: MessageEvent<string>) => void | Promise<void>;
  createSource?: (url: string) => EventSourceClient;
  setTimer?: (callback: () => void, delay: number) => ReturnType<typeof setTimeout>;
  clearTimer?: (timer: ReturnType<typeof setTimeout>) => void;
  logger?: SSELogger;
  establishmentTimeout?: number;
  initialRetryDelay?: number;
  maximumRetryDelay?: number;
}

const DEFAULT_ESTABLISHMENT_TIMEOUT = 10000;
const DEFAULT_INITIAL_RETRY_DELAY = 1000;
const DEFAULT_MAXIMUM_RETRY_DELAY = 30000;

export class SSEController {
  private readonly url: string;
  private readonly onMessage: (event: MessageEvent<string>) => void | Promise<void>;
  private readonly createSource: (url: string) => EventSourceClient;
  private readonly setTimer: (callback: () => void, delay: number) => ReturnType<typeof setTimeout>;
  private readonly clearTimer: (timer: ReturnType<typeof setTimeout>) => void;
  private readonly logger: SSELogger;
  private readonly establishmentTimeout: number;
  private readonly initialRetryDelay: number;
  private readonly maximumRetryDelay: number;

  private source: EventSourceClient | null = null;
  private watchdogTimer: ReturnType<typeof setTimeout> | null = null;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private generation = 0;
  private retryCount = 0;
  private disposed = false;

  constructor(options: SSEControllerOptions) {
    this.url = options.url;
    this.onMessage = options.onMessage;
    this.createSource = options.createSource ?? ((url) => new EventSource(url));
    this.setTimer = options.setTimer ?? ((cb, delay) => setTimeout(cb, delay));
    this.clearTimer = options.clearTimer ?? ((t) => clearTimeout(t));
    this.logger = options.logger ?? console;
    this.establishmentTimeout = options.establishmentTimeout ?? DEFAULT_ESTABLISHMENT_TIMEOUT;
    this.initialRetryDelay = options.initialRetryDelay ?? DEFAULT_INITIAL_RETRY_DELAY;
    this.maximumRetryDelay = options.maximumRetryDelay ?? DEFAULT_MAXIMUM_RETRY_DELAY;
  }

  start() {
    if (this.disposed || this.source) return;
    this.connect();
  }

  dispose() {
    if (this.disposed) return;

    this.disposed = true;
    this.generation++;
    this.clearWatchdog();
    this.clearRetry();
    this.source?.close();
    this.source = null;
    this.logger.info('Sentinel SSE disposed');
  }

  private connect() {
    if (this.disposed) return;

    const source = this.createSource(this.url);
    const generation = ++this.generation;
    this.source = source;
    this.watchdogTimer = this.setTimer(() => {
      if (!this.isActive(source, generation)) return;

      this.watchdogTimer = null;
      source.close();
      this.source = null;
      this.logger.warn('Sentinel SSE connection timed out');
      this.scheduleRetry('timeout');
    }, this.establishmentTimeout);

    source.addEventListener('message', (event) => {
      if (!this.isActive(source, generation)) return;
      void this.onMessage(event);
    });

    source.addEventListener('open', () => {
      if (!this.isActive(source, generation)) return;

      this.clearWatchdog();
      this.retryCount = 0;
      this.logger.info('Sentinel SSE connection established');
    });

    source.addEventListener('error', () => {
      if (!this.isActive(source, generation)) return;

      this.clearWatchdog();
      source.close();
      this.source = null;
      this.logger.warn('Sentinel SSE connection error');
      this.scheduleRetry('error');
    });
  }

  private scheduleRetry(reason: 'error' | 'timeout') {
    if (this.disposed || this.retryTimer) return;

    const delay = Math.min(this.initialRetryDelay * Math.pow(2, this.retryCount), this.maximumRetryDelay);
    this.retryCount++;
    this.logger.info(`Sentinel SSE retry scheduled after ${reason} in ${delay}ms`);
    this.retryTimer = this.setTimer(() => {
      this.retryTimer = null;
      this.connect();
    }, delay);
  }

  private isActive(source: EventSourceClient, generation: number) {
    return !this.disposed && this.source === source && this.generation === generation;
  }

  private clearWatchdog() {
    if (!this.watchdogTimer) return;
    this.clearTimer(this.watchdogTimer);
    this.watchdogTimer = null;
  }

  private clearRetry() {
    if (!this.retryTimer) return;
    this.clearTimer(this.retryTimer);
    this.retryTimer = null;
  }
}
