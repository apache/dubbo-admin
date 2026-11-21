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

import { describe, expect, it } from 'vitest'
import { formItemToTag, tagToFormItem, type Tag } from './tagFormItem'

describe('tag rule form mapping', () => {
  it('keeps the addresses of a tag across an edit round-trip', () => {
    const tag: Tag = { name: 'gray', match: [], addresses: ['0.0.0.0:20001', '0.0.0.0:20002'] }
    const item = tagToFormItem(tag)
    expect(item.scope.type).toBe('addresses')
    expect(formItemToTag(item)).toEqual(tag)
  })

  it('keeps labels and addresses of a tag that has both', () => {
    const tag: Tag = {
      name: 'gray',
      match: [{ key: 'env', value: { exact: 'gray' } }],
      addresses: ['0.0.0.0:20001']
    }
    const item = tagToFormItem(tag)
    expect(item.scope.type).toBe('labels')
    expect(formItemToTag(item)).toEqual(tag)
  })

  it('opens a tag without match', () => {
    expect(() => tagToFormItem({ name: 'gray', addresses: ['0.0.0.0:20001'] })).not.toThrow()
  })

  it('splits addresses typed with either comma', () => {
    const item = tagToFormItem({ name: 'gray' })
    item.scope.addresses.addressesStr = '0.0.0.0:20001， 0.0.0.0:20002,'
    expect(formItemToTag(item).addresses).toEqual(['0.0.0.0:20001', '0.0.0.0:20002'])
  })

  it('drops the labels hidden by the address editor', () => {
    const item = tagToFormItem({ name: 'gray', match: [{ key: 'env', value: { exact: 'gray' } }] })
    item.scope.type = 'addresses' // what deleting the last label does
    item.scope.addresses.addressesStr = '10.0.0.1:20880'
    expect(formItemToTag(item)).toEqual({ name: 'gray', match: [], addresses: ['10.0.0.1:20880'] })
  })

  it('omits addresses that were not filled in', () => {
    expect(formItemToTag(tagToFormItem({ name: 'gray' }))).toEqual({ name: 'gray', match: [] })
  })
})
