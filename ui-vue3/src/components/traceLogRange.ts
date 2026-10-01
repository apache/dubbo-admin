const traceIdPattern = /^(?:[0-9a-fA-F]{32}|[0-9a-fA-F]{16})$/

export function isValidTraceId(value: string): boolean {
  return traceIdPattern.test(value)
}

export type TraceLogRangeInput = {
  traceId?: string
  startTime?: string | number
}

export type TraceLogRange = {
  traceId: string
  from: number
  to: number
}

export function resolveTraceLogRange(input: TraceLogRangeInput): TraceLogRange {
  const traceId = String(input.traceId || '').trim()
  if (!isValidTraceId(traceId)) {
    throw new Error('A valid 16- or 32-character hexadecimal Trace ID is required')
  }

  const startTime = Number(input.startTime)
  if (!Number.isFinite(startTime) || startTime <= 0) {
    throw new Error('A valid Trace start time is required')
  }
  return { traceId, from: startTime - 300000, to: startTime + 300000 }
}
