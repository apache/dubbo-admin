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

// @vitest-environment node

import { setupServer } from 'msw/node'
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import type { ApiResponse, ServiceMethod, ServiceMethodDetail } from '@/types/api'
import { serviceHandlers } from './service'

const origin = 'http://localhost'
const server = setupServer(...serviceHandlers)
const serviceParams = {
  mesh: 'default',
  serviceName: 'org.apache.dubbo.samples.UserService',
  group: 'group1',
  version: '1.0.0'
}
const stringSignature = 'java.lang.String->org.apache.demo.UserDTO'
const charSignature = 'char->org.apache.demo.UserDTO'
const createSignature = 'org.apache.demo.UserCreateReq->java.lang.Boolean'
const deleteSignature = 'java.lang.String->java.lang.Boolean'

beforeAll(() => {
  // Resolve the production handlers' relative URLs, as the browser does in mock mode.
  vi.stubGlobal('location', new URL(origin))
  server.listen({ onUnhandledRequest: 'error' })
})
afterEach(() => server.resetHandlers())
afterAll(() => {
  server.close()
  vi.unstubAllGlobals()
})

async function get<T>(path: string, params: Record<string, string> = {}) {
  const url = new URL(`/api/v1/service/${path}`, origin)
  url.search = new URLSearchParams({ ...serviceParams, ...params }).toString()
  const response = await fetch(url)
  expect(response.status).toBe(200)
  const body = (await response.json()) as ApiResponse<T>
  return body
}

async function invoke(methodName?: string, signature?: string, args: unknown[] = []) {
  const response = await fetch(`${origin}/api/v1/service/generic/invoke`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      ...serviceParams,
      instanceName: 'dubbo-provider-0',
      methodName,
      signature,
      args,
      timeoutMs: 3000
    })
  })
  expect(response.status).toBe(200)
  const body = (await response.json()) as ApiResponse<{ elapsedMs: number; rawResult: unknown }>
  return body
}

describe('service mock method/signature matching', () => {
  it('lists both query overloads with distinct signatures', async () => {
    const body = await get<ServiceMethod[]>('methods')
    expect(body.code).toBe('Success')
    expect(body.data.filter((method) => method.methodName === 'query')).toEqual(
      expect.arrayContaining([
        { methodName: 'query', parameterTypes: ['java.lang.String'], signature: stringSignature },
        { methodName: 'query', parameterTypes: ['char'], signature: charSignature }
      ])
    )
  })

  it.each([
    ['query', stringSignature, 'java.lang.String', 'org.apache.demo.UserDTO'],
    ['query', charSignature, 'char', 'org.apache.demo.UserDTO'],
    ['create', createSignature, 'org.apache.demo.UserCreateReq', 'java.lang.Boolean'],
    ['delete', deleteSignature, 'java.lang.String', 'java.lang.Boolean']
  ])(
    'returns matching detail for %s / %s',
    async (methodName, signature, parameterType, returnType) => {
      const body = await get<ServiceMethodDetail>('method/detail', { methodName, signature })
      expect(body.code).toBe('Success')
      expect(body.data).toMatchObject({
        methodName,
        signature,
        parameterTypes: [parameterType],
        parameters: [{ type: parameterType }],
        returnType
      })
      if (returnType === 'org.apache.demo.UserDTO') {
        expect(body.data.types).toContainEqual({
          type: returnType,
          properties: {
            id: 'java.lang.String',
            name: 'java.lang.String',
            age: 'java.lang.Integer'
          },
          items: [],
          enums: []
        })
      }
    }
  )

  // The char overload is a target fixture, absent from the RED baseline.
  // Give it a distinct UserDTO so a name-only lookup cannot accidentally pass.
  it.each([
    {
      methodName: 'query',
      signature: stringSignature,
      args: ['1001'],
      result: { id: '1001', name: 'Alice', age: 18 }
    },
    {
      methodName: 'query',
      signature: charSignature,
      args: ['a'],
      result: { id: '1002', name: 'Bob', age: 20 }
    },
    {
      methodName: 'create',
      signature: createSignature,
      args: [{ name: 'Carol', age: 21, email: 'carol@example.com' }],
      result: true
    },
    { methodName: 'delete', signature: deleteSignature, args: ['1001'], result: true }
  ])(
    'invokes $methodName / $signature with the declared result',
    async ({ methodName, signature, args, result }) => {
      const body = await invoke(methodName, signature, args)
      expect(body.code).toBe('Success')
      expect(body.data.elapsedMs).toEqual(expect.any(Number))
      expect(body.data.rawResult).toEqual(result)
    }
  )

  describe.each([
    { scenario: 'unknown method', methodName: 'missingMethod', signature: stringSignature },
    {
      scenario: 'unknown signature',
      methodName: 'query',
      signature: 'int->org.apache.demo.UserDTO',
      expected: `${charSignature} | ${stringSignature}`
    },
    {
      scenario: 'another method signature',
      methodName: 'query',
      signature: createSignature,
      expected: `${charSignature} | ${stringSignature}`
    },
    {
      scenario: 'query signature with create',
      methodName: 'create',
      signature: stringSignature,
      expected: createSignature
    }
  ])('rejects $scenario without a successful fallback', ({ methodName, signature, expected }) => {
    const error = {
      code: 'NotFoundError',
      message: `method ${methodName} not found for service ${serviceParams.serviceName}`,
      data: null
    }

    it('rejects method detail lookup', async () => {
      expect(await get('method/detail', { methodName, signature })).toEqual(error)
    })

    it('rejects generic invoke', async () => {
      expect(await invoke(methodName, signature)).toEqual({
        ...error,
        ...(expected ? { field: 'signature', expected, actual: signature } : {})
      })
    })
  })

  describe.each([
    { scenario: 'omitted', value: undefined },
    { scenario: 'empty', value: '' },
    { scenario: 'whitespace-only', value: ' \t\n ' }
  ])('$scenario required fields', ({ value }) => {
    it.each(['methodName', 'signature'] as const)('rejects method detail %s', async (field) => {
      const params: Record<string, string> = { methodName: 'query', signature: stringSignature }
      if (value === undefined) delete params[field]
      else params[field] = value
      expect(await get('method/detail', params)).toEqual({
        code: 'InvalidArgument',
        message: field === 'methodName' ? 'method name is empty' : 'signature is empty',
        data: null
      })
    })

    it.each(['methodName', 'signature'] as const)(
      'rejects invoke %s with field details',
      async (field) => {
        const body = await invoke(
          field === 'methodName' ? value : 'query',
          field === 'signature' ? value : stringSignature
        )
        expect(body).toEqual({
          code: 'InvalidArgument',
          message: field === 'methodName' ? 'method name is empty' : 'signature is empty',
          data: null,
          field,
          expected: 'non-empty',
          actual: ''
        })
      }
    )
  })

  it('validates method name before signature for method detail', async () => {
    expect(await get('method/detail')).toEqual({
      code: 'InvalidArgument',
      message: 'method name is empty',
      data: null
    })
  })

  it('validates method name before signature for invoke', async () => {
    expect(await invoke()).toEqual({
      code: 'InvalidArgument',
      message: 'method name is empty',
      data: null,
      field: 'methodName',
      expected: 'non-empty',
      actual: ''
    })
  })

  it.each([
    { signature: stringSignature, args: ['1001'], result: { id: '1001', name: 'Alice', age: 18 } },
    { signature: charSignature, args: ['a'], result: { id: '1002', name: 'Bob', age: 20 } }
  ])('trims whitespace when selecting $signature', async ({ signature, args, result }) => {
    const detail = await get<ServiceMethodDetail>('method/detail', {
      methodName: ' \tquery\n ',
      signature: ` \t${signature}\n `
    })
    expect(detail.code).toBe('Success')
    expect(detail.data.signature).toBe(signature)
    const body = await invoke(' \tquery\n ', ` \t${signature}\n `, args)
    expect(body.code).toBe('Success')
    expect(body.data.rawResult).toEqual(result)
  })

  it('reports the trimmed signature in invoke mismatch errors', async () => {
    expect(await invoke(' query ', ' int->org.apache.demo.UserDTO ')).toEqual({
      code: 'NotFoundError',
      message: `method query not found for service ${serviceParams.serviceName}`,
      data: null,
      field: 'signature',
      expected: `${charSignature} | ${stringSignature}`,
      actual: 'int->org.apache.demo.UserDTO'
    })
  })

  it.each(['', '{'])('returns InvalidArgument for malformed JSON %j', async (body) => {
    const response = await fetch(`${origin}/api/v1/service/generic/invoke`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body
    })
    expect(response.status).toBe(200)
    expect(await response.json()).toEqual({
      code: 'InvalidArgument',
      message: 'invalid JSON request body',
      data: null
    })
  })
})
