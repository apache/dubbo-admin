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

export interface ConditionRuleDestination {
  match: string
  weight: number
}

export interface StructuredConditionRule {
  from: { match: string }
  to: ConditionRuleDestination[]
}

export const newStructuredConditionRule = (): StructuredConditionRule => ({
  from: { match: '' },
  to: [{ match: '', weight: 0 }]
})

export const normalizeStructuredConditions = (conditions: unknown): StructuredConditionRule[] => {
  if (!Array.isArray(conditions)) {
    return []
  }
  return conditions.map((condition: any) => ({
    from: { match: String(condition?.from?.match || '') },
    to: Array.isArray(condition?.to)
      ? condition.to.map((destination: any) => ({
          match: String(destination?.match || ''),
          weight: Number(destination?.weight ?? 0)
        }))
      : []
  }))
}

export const isCompleteConditionRule = (data: unknown): data is Record<string, any> => {
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    return false
  }
  return [
    'configVersion',
    'priority',
    'enabled',
    'force',
    'runtime',
    'key',
    'scope',
    'conditions'
  ].every((field) => Object.prototype.hasOwnProperty.call(data, field))
}
