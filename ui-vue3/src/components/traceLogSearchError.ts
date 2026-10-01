export function traceLogSearchErrorMessage(error: unknown): string {
  const code = (error as { code?: unknown } | null)?.code
  switch (code) {
    case 'InvalidArgument':
      return 'Trace ID must be 16 or 32 hexadecimal characters'
    case 'NotFoundError':
      return 'Trace does not exist or has no valid start time'
    case 'ConfigError':
      return 'Jaeger query is not configured'
    case 'NetWorkError':
      return 'Jaeger query is unavailable. Please try again later'
    default:
      return 'Unable to retrieve the Trace time range'
  }
}

export function traceLogDashboardErrorMessage(error: unknown): string {
  const code = (error as { code?: unknown } | null)?.code
  if (code === 'NotFoundError' || code === 'ConfigError') {
    return 'Trace log dashboard is not configured'
  }
  return 'Log dashboard is unavailable. Please try again later'
}
