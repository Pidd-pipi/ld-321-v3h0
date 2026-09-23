export const logger = {
  debug: (...args: unknown[]) => console.debug('[agridispatch]', ...args),
  info: (...args: unknown[]) => console.info('[agridispatch]', ...args),
  warn: (...args: unknown[]) => console.warn('[agridispatch]', ...args),
  error: (...args: unknown[]) => console.error('[agridispatch]', ...args),
};
