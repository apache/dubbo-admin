/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. */
import { AIAPIError } from '@/api/service/aiMemory'

const chatErrors = new Set([
  'SESSION_BUSY',
  'INTERACTION_IN_PROGRESS',
  'IDEMPOTENCY_CONFLICT',
  'RESOURCE_EXHAUSTED',
  'AUTHENTICATION_REQUIRED',
  'RESOURCE_NOT_ACCESSIBLE'
])

export function chatErrorKey(error: unknown): string | undefined {
  if (!(error instanceof AIAPIError)) return undefined
  const code = chatErrors.has(error.code) ? error.code : 'SERVER_ERROR'
  return `aiChat.errors.${code}`
}
