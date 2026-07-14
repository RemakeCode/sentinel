const SENTINEL_BADGE = '%cSentinel%c';
const BADGE_STYLE =
  'background:#c2410c;color:#ffffff;padding:1px 5px;border-radius:3px;font-weight:bold;margin-right:4px;';
const RESET_STYLE = '';

type LogMethod = (...args: unknown[]) => void;

export interface SentinelLogger {
  log: LogMethod;
  info: LogMethod;
  warn: LogMethod;
  error: LogMethod;
  debug: LogMethod;
}

function makeLogMethod(consoleMethod: LogMethod): LogMethod {
  return (...args: unknown[]) => consoleMethod(SENTINEL_BADGE, BADGE_STYLE, RESET_STYLE, ...args);
}

export const sentinelLogger: SentinelLogger = {
  log: makeLogMethod(console.log),
  info: makeLogMethod(console.info),
  warn: makeLogMethod(console.warn),
  error: makeLogMethod(console.error),
  debug: makeLogMethod(console.debug),
};
