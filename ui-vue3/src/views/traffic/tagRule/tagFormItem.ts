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

export interface TagMatch {
  key: string
  value: Record<string, string>
}

export interface Tag {
  name: string
  match?: TagMatch[]
  addresses?: string[]
}

export interface TagLabelItem {
  myKey: string
  condition: string
  value: string
}

export interface TagFormItem {
  tagName: string
  scope: {
    type: 'labels' | 'addresses'
    labels: TagLabelItem[]
    addresses: { condition: string; addressesStr: string }
  }
}

// tagToFormItem maps a tag of a tag rule to the edit form. A tag that only
// selects addresses opens with the address editor.
export const tagToFormItem = (tag: Tag): TagFormItem => {
  const labels = (tag.match ?? []).map((m) => {
    const condition = Object.keys(m.value)[0]
    return { myKey: m.key, condition, value: m.value[condition] }
  })
  const addresses = tag.addresses ?? []
  return {
    tagName: tag.name,
    scope: {
      type: addresses.length > 0 && labels.length === 0 ? 'addresses' : 'labels',
      labels,
      addresses: { condition: '=', addressesStr: addresses.join(',') }
    }
  }
}

// formItemToTag maps an edit form item back to a tag. Addresses are kept when
// present. Labels are kept unless the form selects by address: deleting the
// last label switches to the address editor and only hides that label, and a
// label match would take precedence over the addresses.
export const formItemToTag = (item: TagFormItem): Tag => {
  const labels = item.scope.type === 'labels' ? item.scope.labels : []
  const tag: Tag = {
    name: item.tagName,
    match: labels.map((l) => ({ key: l.myKey, value: { [l.condition]: l.value } }))
  }
  const addresses = item.scope.addresses.addressesStr
    .split(/[,，]/)
    .map((a) => a.trim())
    .filter((a) => a)
  if (addresses.length > 0) {
    tag.addresses = addresses
  }
  return tag
}
