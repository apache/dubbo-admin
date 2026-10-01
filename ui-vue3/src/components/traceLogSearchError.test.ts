import { describe, expect, it } from 'vitest'
import { traceLogDashboardErrorMessage, traceLogSearchErrorMessage } from './traceLogSearchError'

describe('traceLogSearchErrorMessage', () => {
  it('does not report an invalid Trace ID as a missing trace', () => {
    expect(traceLogSearchErrorMessage({ code: 'InvalidArgument' })).toBe(
      'Trace ID must be 16 or 32 hexadecimal characters'
    )
  })

  it('distinguishes a missing trace from configuration and network errors', () => {
    expect(traceLogSearchErrorMessage({ code: 'NotFoundError' })).toContain('does not exist')
    expect(traceLogSearchErrorMessage({ code: 'ConfigError' })).toContain('not configured')
    expect(traceLogSearchErrorMessage({ code: 'NetWorkError' })).toContain('unavailable')
  })

  it('uses a neutral message for an unexpected failure', () => {
    expect(traceLogSearchErrorMessage(new Error('unexpected'))).toBe(
      'Unable to retrieve the Trace time range'
    )
  })
})

describe('traceLogDashboardErrorMessage', () => {
  it('reports a missing dashboard without blaming the Trace lookup', () => {
    expect(traceLogDashboardErrorMessage({ code: 'NotFoundError' })).toContain('not configured')
  })

  it('reports dashboard transport failures separately from Jaeger failures', () => {
    expect(traceLogDashboardErrorMessage(undefined)).toContain('Log dashboard is unavailable')
  })
})
