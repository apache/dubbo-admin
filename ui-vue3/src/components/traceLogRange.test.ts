import { describe, expect, it } from 'vitest'
import { isValidTraceId, resolveTraceLogRange } from './traceLogRange'

describe('resolveTraceLogRange', () => {
  const traceId = '766b9604d84bd8e6b17b2292dfd9bdd0'

  it('accepts only complete 16- or 32-character hexadecimal Trace IDs', () => {
    expect(isValidTraceId('766b9604d84bd8e6')).toBe(true)
    expect(isValidTraceId(traceId)).toBe(true)
    expect(isValidTraceId('6565656')).toBe(false)
    expect(isValidTraceId(`${traceId}x`)).toBe(false)
  })

  it('derives a five-minute range from startTime', () => {
    expect(resolveTraceLogRange({ traceId, startTime: 1790774974677 })).toEqual({
      traceId,
      from: 1790774674677,
      to: 1790775274677
    })
  })

  it('rejects invalid startTime', () => {
    expect(() => resolveTraceLogRange({ traceId, startTime: 'invalid' })).toThrow(
      'valid Trace start time'
    )
  })

  it('rejects missing time state', () => {
    expect(() => resolveTraceLogRange({ traceId })).toThrow('valid Trace start time')
  })
})
