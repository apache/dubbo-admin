/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
