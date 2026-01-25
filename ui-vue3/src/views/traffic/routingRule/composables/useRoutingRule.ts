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

import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

// Type definitions
export interface ConditionItem {
  type: string
  condition?: string
  value?: string
  list?: Array<Record<string, any>>
}

interface ParseOptions {
  availableTypes: readonly string[]
  isMatchCondition: boolean
}

interface CommonRouteItem {
  selectedMatchConditionTypes: string[]
  requestMatch: ConditionItem[]
  selectedRouteDistributeMatchTypes: string[]
  routeDistribute: ConditionItem[]
}

// Condition type configuration
export const CONDITION_TYPE_CONFIG = {
  // Single value types: type=value or type!=value
  single: ['host', 'application', 'method'],
  // Array types: type[key]=value or type[key]!=value
  array: ['arguments', 'attachments'],
  // Custom types: key=value (without type prefix)
  custom: ['other']
} as const

export default function useRoutingRule() {
  const { t } = useI18n()

  const matchConditionTypeOptions = computed(() => [
    { label: 'host', value: 'host' },
    { label: 'application', value: 'application' },
    { label: 'method', value: 'method' },
    { label: 'arguments', value: 'arguments' },
    { label: 'attachments', value: 'attachments' },
    { label: t('routingRuleDomain.other'), value: 'other' }
  ])

  const routeDistributionTypeOptions = computed(() => [
    { label: 'host', value: 'host' },
    { label: t('routingRuleDomain.other'), value: 'other' }
  ])

  const conditionOptions = computed(() => [
    { label: '=', value: '=' },
    { label: '!=', value: '!=' }
  ])

  const routeList = ref<CommonRouteItem[]>([
    {
      selectedMatchConditionTypes: [],
      requestMatch: [],
      selectedRouteDistributeMatchTypes: [],
      routeDistribute: [
        { type: 'host', condition: '', value: '' },
        { type: 'other', list: [] }
      ]
    }
  ])

  // --- Parsing Logic ---

  function parseConditionPart(part: string, resultArray: ConditionItem[], type: string): boolean {
    part = part.trim()

    // Handle single value types (host, application, method)
    if (CONDITION_TYPE_CONFIG.single.includes(type as any)) {
      const match = part.match(new RegExp(`^${type}(!=|=)(.+)`))
      if (match) {
        resultArray.push({ type, condition: match[1], value: match[2].trim() })
        return true
      }
    }

    // Handle arguments: arguments[index]=value
    if (type === 'arguments') {
      const match = part.match(/^arguments\[(\d+)\](!=|=)(.+)/)
      if (match) {
        let argObj = resultArray.find((item) => item.type === 'arguments')
        if (!argObj) {
          argObj = { type: 'arguments', list: [] }
          resultArray.push(argObj)
        }
        argObj.list!.push({
          index: parseInt(match[1], 10),
          condition: match[2],
          value: match[3].trim()
        })
        return true
      }
    }

    // Handle attachments: attachments[key]=value
    if (type === 'attachments') {
      const match = part.match(/^attachments\[(.+)\](!=|=)(.+)/)
      if (match) {
        let attachObj = resultArray.find((item) => item.type === 'attachments')
        if (!attachObj) {
          attachObj = { type: 'attachments', list: [] }
          resultArray.push(attachObj)
        }
        attachObj.list!.push({
          myKey: match[1].trim(),
          condition: match[2],
          value: match[3].trim()
        })
        return true
      }
    }

    return false
  }

  function parseConditionString(
    conditionStr: string,
    routeItemIndex: number,
    options: ParseOptions
  ): ConditionItem[] {
    const { availableTypes, isMatchCondition } = options
    const resultArray: ConditionItem[] = []
    const selectedTypesKey = isMatchCondition
      ? 'selectedMatchConditionTypes'
      : 'selectedRouteDistributeMatchTypes'

    // Clear selected types for this route item
    if (routeList.value[routeItemIndex]) {
      routeList.value[routeItemIndex][selectedTypesKey] = []
    }

    if (!conditionStr) {
      // Return default empty structure
      return availableTypes.map((type) => {
        if (
          CONDITION_TYPE_CONFIG.array.includes(type as any) ||
          CONDITION_TYPE_CONFIG.custom.includes(type as any)
        ) {
          return { type, list: [] }
        }
        return { type, condition: '', value: '' }
      })
    }

    const parts = conditionStr.split(' & ')

    parts.forEach((part) => {
      if (!part.trim()) return

      let parsed = false

      // Try to parse with known types
      for (const type of availableTypes) {
        if (part.startsWith(type)) {
          if (parseConditionPart(part, resultArray, type)) {
            // Add to selected types if not already present
            const selectedTypes = routeList.value[routeItemIndex][selectedTypesKey]
            if (!selectedTypes.includes(type)) {
              selectedTypes.push(type)
            }
            parsed = true
            break
          }
        }
      }

      // Handle custom key=value pairs (other type)
      if (!parsed) {
        const match = part.match(/^([^!=]+)(!?=)(.+)$/)
        if (match) {
          const type = 'other'
          let otherObj = resultArray.find((item) => item.type === type)
          if (!otherObj) {
            otherObj = { type, list: [] }
            resultArray.push(otherObj)
          }
          otherObj.list!.push({
            myKey: match[1].trim(),
            condition: match[2],
            value: match[3].trim()
          })

          // Add to selected types
          const selectedTypes = routeList.value[routeItemIndex][selectedTypesKey]
          if (!selectedTypes.includes(type)) {
            selectedTypes.push(type)
          }
        }
      }
    })

    // Add default empty structures for types that weren't found
    availableTypes.forEach((type) => {
      if (!resultArray.find((item) => item.type === type)) {
        if (
          CONDITION_TYPE_CONFIG.array.includes(type as any) ||
          CONDITION_TYPE_CONFIG.custom.includes(type as any)
        ) {
          resultArray.push({ type, list: [] })
        } else {
          resultArray.push({ type, condition: '', value: '' })
        }
      }
    })

    return resultArray
  }

  function parseConditionMatchStringToArray(matchStr: string, routeItemIndex: number) {
    return parseConditionString(matchStr, routeItemIndex, {
      availableTypes: [
        ...CONDITION_TYPE_CONFIG.single,
        ...CONDITION_TYPE_CONFIG.array,
        ...CONDITION_TYPE_CONFIG.custom
      ],
      isMatchCondition: true
    })
  }

  function parseConditionToStringToArray(toStr: string, routeItemIndex: number) {
    return parseConditionString(toStr || '', routeItemIndex, {
      availableTypes: ['host', 'other'],
      isMatchCondition: false
    })
  }

  // --- Merging Logic ---

  function mergeConditionItems(
    selectedTypes: string[],
    conditionItems: any[],
    separator: string = ' & '
  ): string {
    const result: string[] = []

    selectedTypes.forEach((type) => {
      const item = conditionItems.find((i) => i.type === type)
      if (!item) return

      // Handle list-based types (arguments, attachments, other)
      if (item.list && Array.isArray(item.list)) {
        item.list.forEach((listItem: any) => {
          if (listItem.value !== undefined && listItem.value !== '') {
            if (type === 'arguments') {
              result.push(`${type}[${listItem.index}]${listItem.condition}${listItem.value}`)
            } else if (type === 'attachments') {
              result.push(`${type}[${listItem.myKey}]${listItem.condition}${listItem.value}`)
            } else if (type === 'other') {
              result.push(`${listItem.myKey}${listItem.condition}${listItem.value}`)
            }
          }
        })
      }
      // Handle single value types (host, application, method)
      else if (item.value !== undefined && item.value !== '') {
        result.push(`${item.type}${item.condition}${item.value}`)
      }
    })

    return result.join(separator)
  }

  function mergeConditions() {
    const conditions: string[] = []

    routeList.value.forEach((routeItem) => {
      // Merge match conditions (when)
      const matchStr = mergeConditionItems(
        routeItem.selectedMatchConditionTypes,
        routeItem.requestMatch
      )

      // Merge distribute conditions (then)
      const toStr = mergeConditionItems(
        routeItem.selectedRouteDistributeMatchTypes,
        routeItem.routeDistribute
      )

      // Only add condition if there's actual content in match part
      if (matchStr.length > 0) {
        if (toStr.length > 0) {
          conditions.push(`${matchStr} => ${toStr}`)
        } else {
          conditions.push(matchStr)
        }
      }
    })

    return conditions
  }

  // --- List Manipulation ---

  const addRoute = () => {
    routeList.value.push({
      selectedMatchConditionTypes: [],
      requestMatch: [],
      selectedRouteDistributeMatchTypes: [],
      routeDistribute: [
        { type: 'host', condition: '', value: '' },
        { type: 'other', list: [] }
      ]
    })
  }

  const deleteRoute = (index: number) => {
    routeList.value.splice(index, 1)
  }

  const deleteRequestMatch = (index: number) => {
    routeList.value[index].requestMatch = []
    routeList.value[index].selectedMatchConditionTypes = []
  }

  const addRequestMatch = (index: number) => {
    routeList.value[index].requestMatch = [
      { type: 'host', condition: '', value: '' },
      { type: 'application', condition: '', value: '' },
      { type: 'method', condition: '', value: '' },
      { type: 'arguments', list: [] },
      { type: 'attachments', list: [] },
      { type: 'other', list: [] }
    ]
  }

  const deleteMatchConditionTypeItem = (type: string, index: number) => {
    routeList.value[index].selectedMatchConditionTypes = routeList.value[
      index
    ].selectedMatchConditionTypes.filter((item) => item !== type)
  }

  const deleteRouteDistributeMatchTypeItem = (type: string, index: number) => {
    routeList.value[index].selectedRouteDistributeMatchTypes = routeList.value[
      index
    ].selectedRouteDistributeMatchTypes.filter((item) => item !== type)
  }

  // --- Item Manipulation ---

  const addArgumentsItem = (routeItemIndex: number, conditionItemIndex: number) => {
    routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list!.push({
      index: 0,
      condition: '=',
      value: ''
    })
  }

  const deleteArgumentsItem = (
    routeItemIndex: number,
    conditionItemIndex: number,
    argumentsIndex: number
  ) => {
    if (routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list!.length === 1) {
      routeList.value[routeItemIndex].selectedMatchConditionTypes = routeList.value[
        routeItemIndex
      ].selectedMatchConditionTypes.filter((item) => item !== 'arguments')
    }
    routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list!.splice(argumentsIndex, 1)
  }

  const addAttachmentsItem = (routeItemIndex: number, conditionItemIndex: number) => {
    routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list!.push({
      myKey: '',
      condition: '=',
      value: ''
    })
  }

  const deleteAttachmentsItem = (
    routeItemIndex: number,
    conditionItemIndex: number,
    attachmentsItemIndex: number
  ) => {
    if (routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list!.length === 1) {
      routeList.value[routeItemIndex].selectedMatchConditionTypes = routeList.value[
        routeItemIndex
      ].selectedMatchConditionTypes.filter((item) => item !== 'attachments')
    }
    routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list!.splice(
      attachmentsItemIndex,
      1
    )
  }

  const addOtherItem = (routeItemIndex: number, conditionItemIndex: number) => {
    routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list!.push({
      myKey: '',
      condition: '=',
      value: ''
    })
  }

  const deleteOtherItem = (
    routeItemIndex: number,
    conditionItemIndex: number,
    otherItemIndex: number
  ) => {
    if (routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list!.length === 1) {
      routeList.value[routeItemIndex].selectedMatchConditionTypes = routeList.value[
        routeItemIndex
      ].selectedMatchConditionTypes.filter((item) => item !== 'other')
      return
    }
    routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list!.splice(otherItemIndex, 1)
  }

  const addRouteDistributeOtherItem = (routeItemIndex: number, conditionItemIndex: number) => {
    routeList.value[routeItemIndex].routeDistribute[conditionItemIndex].list!.push({
      myKey: '',
      condition: '=',
      value: ''
    })
  }

  const deleteRouteDistributeOtherItem = (
    routeItemIndex: number,
    conditionItemIndex: number,
    otherItemIndex: number
  ) => {
    if (routeList.value[routeItemIndex].routeDistribute[conditionItemIndex].list!.length === 1) {
      routeList.value[routeItemIndex].selectedRouteDistributeMatchTypes = routeList.value[
        routeItemIndex
      ].selectedRouteDistributeMatchTypes.filter((item) => item !== 'other')
      return
    }
    routeList.value[routeItemIndex].routeDistribute[conditionItemIndex].list!.splice(
      otherItemIndex,
      1
    )
  }

  // --- Description Logic ---

  const routeItemDes = (
    routeIndex: number,
    baseInfo: { ruleGranularity: string; objectOfAction: string }
  ) => {
    const routeItem = routeList.value[routeIndex]
    const { ruleGranularity, objectOfAction } = baseInfo

    const typeText =
      ruleGranularity === 'service'
        ? t('routingRuleDomain.service')
        : t('routingRuleDomain.application')
    const baseDescription = t('routingRuleDomain.baseDesc', {
      type: typeText,
      value: objectOfAction
    })

    // 构建匹配条件描述 (when)
    const whenConditions: string[] = []
    routeItem.selectedMatchConditionTypes?.forEach((type) => {
      const matchItem = routeItem.requestMatch?.find((item) => item.type === type)
      if (!matchItem) return

      let conditionStr = ''

      const relation = matchItem.condition
      const val = matchItem.value || t('serviceDomain.notSpecified')

      switch (type) {
        case 'host':
          conditionStr = t('routingRuleDomain.matchDesc', {
            condition: `IP ${relation} ${val}`
          })
          break
        case 'application':
          conditionStr = t('routingRuleDomain.matchDesc', {
            condition: `${t('routingRuleDomain.application')} ${relation} ${val}`
          })
          break
        case 'method':
          conditionStr = t('routingRuleDomain.matchDesc', {
            condition: `${t('routingRuleDomain.method')} ${relation} ${val}`
          })
          break
        case 'arguments': {
          const argConditions = matchItem.list
            ?.map((arg: any) => {
              const argVal =
                arg.value !== undefined && arg.value !== ''
                  ? arg.value
                  : t('serviceDomain.notSpecified')
              return `${t('routingRuleDomain.arguments')}[${arg.index}] ${arg.condition} ${argVal}`
            })
            .filter(Boolean)
          if (argConditions && argConditions.length > 0) conditionStr = argConditions.join(' & ')
          break
        }
        case 'attachments': {
          const attachConditions = matchItem.list
            ?.map((attach: any) => {
              const attachVal =
                attach.value !== undefined && attach.value !== ''
                  ? attach.value
                  : t('serviceDomain.notSpecified')
              return `${t('routingRuleDomain.attachments')}[${attach.myKey || t('serviceDomain.notSpecified')}] ${attach.condition} ${attachVal}`
            })
            .filter(Boolean)
          if (attachConditions && attachConditions.length > 0)
            conditionStr = attachConditions.join(' & ')
          break
        }
        case 'other': {
          const otherConditions = matchItem.list
            ?.map((other: any) => {
              const otherVal =
                other.value !== undefined && other.value !== ''
                  ? other.value
                  : t('serviceDomain.notSpecified')
              return `${t('routingRuleDomain.other')}[${other.myKey || t('serviceDomain.notSpecified')}] ${other.condition} ${otherVal}`
            })
            .filter(Boolean)
          if (otherConditions && otherConditions.length > 0)
            conditionStr = otherConditions.join(' & ')
          break
        }
      }
      if (conditionStr) {
        if ((type === 'host' || type === 'application' || type === 'method') && !matchItem.value) {
          whenConditions.push(`${type} ${t('serviceDomain.notSpecified')}`)
        } else {
          whenConditions.push(conditionStr)
        }
      }
    })

    const whenConditionStr =
      whenConditions.length > 0 ? whenConditions.join(' & ') : t('routingRuleDomain.anyRequest')

    // 构建转发条件描述 (then)
    const thenConditions: string[] = []
    routeItem.selectedRouteDistributeMatchTypes?.forEach((type) => {
      const distributeItem = routeItem.routeDistribute?.find((item) => item.type === type)
      if (!distributeItem) return

      let conditionStr = ''
      const relation = distributeItem.condition
      const val = distributeItem.value || t('serviceDomain.notSpecified')

      switch (type) {
        case 'host':
          conditionStr = t('routingRuleDomain.distributeDesc', {
            condition: `IP ${relation} ${val}`
          })
          break
        case 'other': {
          const otherConditions = distributeItem.list
            ?.map((other: any) => {
              const otherVal =
                other.value !== undefined && other.value !== ''
                  ? other.value
                  : t('serviceDomain.notSpecified')
              return `${t('routingRuleDomain.other')}[${other.myKey || t('serviceDomain.notSpecified')}] ${other.condition} ${otherVal}`
            })
            .filter(Boolean)
          if (otherConditions && otherConditions.length > 0)
            conditionStr = otherConditions.join(' & ')
          break
        }
      }
      if (conditionStr) {
        if (type === 'host' && !distributeItem.value) {
          thenConditions.push(`IP ${t('serviceDomain.notSpecified')}`)
        } else {
          thenConditions.push(conditionStr)
        }
      }
    })

    const thenConditionStr =
      thenConditions.length > 0 ? thenConditions.join(' & ') : t('routingRuleDomain.route')

    return `${baseDescription}, ${t('routingRuleDomain.matchDesc', { condition: whenConditionStr })} -> ${t('routingRuleDomain.distributeDesc', { condition: thenConditionStr })}`
  }

  return {
    matchConditionTypeOptions,
    routeDistributionTypeOptions,
    conditionOptions,
    routeList,
    mergeConditions,
    parseConditionMatchStringToArray,
    parseConditionToStringToArray,
    addRoute,
    deleteRoute,
    deleteRequestMatch,
    addRequestMatch,
    deleteMatchConditionTypeItem,
    deleteRouteDistributeMatchTypeItem,
    addArgumentsItem,
    deleteArgumentsItem,
    addAttachmentsItem,
    deleteAttachmentsItem,
    addOtherItem,
    deleteOtherItem,
    addRouteDistributeOtherItem,
    deleteRouteDistributeOtherItem,
    routeItemDes
  }
}
