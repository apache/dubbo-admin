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

package indexer

import (
	"strings"
	"sync"

	set "github.com/duke-git/lancet/v2/datastructure/set"
)

// RadixNode is a node in the compressed Trie (Radix Tree)
type RadixNode struct {
	// prefix is the string segment stored in this node
	prefix string
	// keys stores all resource keys that end at this node
	keys set.Set[string]
	// children maps the first byte of child prefixes to child nodes
	children map[byte]*RadixNode
	// isLeaf marks whether this node contains actual resource keys
	isLeaf bool
}

// RadixTree is a thread-safe compressed Trie tree for prefix matching
type RadixTree struct {
	root *RadixNode
	mu   sync.RWMutex
	size int
}

// NewRadixTree creates a new compressed Trie tree
func NewRadixTree() *RadixTree {
	return &RadixTree{
		root: &RadixNode{
			prefix:   "",
			keys:     set.New[string](),
			children: make(map[byte]*RadixNode),
			isLeaf:   false,
		},
		size: 0,
	}
}

func (rt *RadixTree) Size() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.size
}

func newNode(prefix string) *RadixNode {
	return &RadixNode{
		prefix:   prefix,
		keys:     set.New[string](),
		children: make(map[byte]*RadixNode),
		isLeaf:   false,
	}
}

func (rt *RadixTree) addKeyToNode(node *RadixNode, resourceKey string) bool {
	if node.keys == nil {
		node.keys = set.New[string]()
	}
	if !node.keys.Contain(resourceKey) {
		node.keys.Add(resourceKey)
		node.isLeaf = true
		rt.size++
		return true
	}
	return false
}

func (node *RadixNode) getChild(value string) (*RadixNode, bool) {
	if value == "" {
		return nil, false
	}
	child, exists := node.children[value[0]]
	return child, exists
}

// Insert inserts an index value and its associated resource key
func (rt *RadixTree) Insert(indexValue, resourceKey string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if indexValue == "" {
		return
	}

	rt.insert(rt.root, indexValue, resourceKey)
}

func (rt *RadixTree) insert(node *RadixNode, indexValue, resourceKey string) {
	if node.prefix == "" || indexValue == "" {
		if indexValue == "" {
			rt.addKeyToNode(node, resourceKey)
			return
		}

		child, exists := node.getChild(indexValue)
		if !exists {
			child = newNode(indexValue)
			child.keys.Add(resourceKey)
			child.isLeaf = true
			node.children[indexValue[0]] = child
			rt.size++
			return
		}

		rt.insert(child, indexValue, resourceKey)
		return
	}

	commonLen := rt.commonPrefixLength(node.prefix, indexValue)

	if commonLen == len(node.prefix) {
		remaining := indexValue[commonLen:]

		if remaining == "" {
			rt.addKeyToNode(node, resourceKey)
			return
		}

		child, exists := node.getChild(remaining)
		if !exists {
			child = newNode(remaining)
			child.keys.Add(resourceKey)
			child.isLeaf = true
			node.children[remaining[0]] = child
			rt.size++
			return
		}

		rt.insert(child, remaining, resourceKey)
		return
	}

	rt.splitNode(node, commonLen, indexValue, resourceKey)
}

func (rt *RadixTree) splitNode(node *RadixNode, commonLen int, indexValue, resourceKey string) {
	commonPrefix := node.prefix[:commonLen]
	oldSuffix := node.prefix[commonLen:]
	newSuffix := indexValue[commonLen:]

	middleNode := newNode(commonPrefix)

	oldChild := &RadixNode{
		prefix:   oldSuffix,
		keys:     node.keys,
		children: node.children,
		isLeaf:   node.isLeaf,
	}
	middleNode.children[oldSuffix[0]] = oldChild

	if newSuffix != "" {
		newChild := newNode(newSuffix)
		newChild.keys.Add(resourceKey)
		newChild.isLeaf = true
		middleNode.children[newSuffix[0]] = newChild
	} else {
		middleNode.keys.Add(resourceKey)
		middleNode.isLeaf = true
	}

	node.prefix = middleNode.prefix
	node.keys = middleNode.keys
	node.children = middleNode.children
	node.isLeaf = middleNode.isLeaf

	rt.size++
}

func (rt *RadixTree) commonPrefixLength(s1, s2 string) int {
	minLen := len(s1)
	if len(s2) < minLen {
		minLen = len(s2)
	}

	for i := 0; i < minLen; i++ {
		if s1[i] != s2[i] {
			return i
		}
	}

	return minLen
}

func (rt *RadixTree) SearchPrefix(prefix string) []string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if prefix == "" {
		return rt.getAllKeys(rt.root)
	}

	node := rt.findPrefixNode(rt.root, prefix)
	if node == nil {
		return []string{}
	}

	return rt.getAllKeys(node)
}

func (rt *RadixTree) findNode(node *RadixNode, value string, exactMatch bool) *RadixNode {
	if value == "" {
		return node
	}

	if node.prefix == "" {
		child, exists := node.getChild(value)
		if !exists {
			return nil
		}
		return rt.findNode(child, value, exactMatch)
	}

	if value == node.prefix {
		return node
	}

	if strings.HasPrefix(value, node.prefix) {
		remaining := value[len(node.prefix):]
		child, exists := node.getChild(remaining)
		if !exists {
			return nil
		}
		return rt.findNode(child, remaining, exactMatch)
	}

	if !exactMatch && strings.HasPrefix(node.prefix, value) {
		return node
	}

	return nil
}

func (rt *RadixTree) findPrefixNode(node *RadixNode, prefix string) *RadixNode {
	return rt.findNode(node, prefix, false)
}

func (rt *RadixTree) findExactNode(node *RadixNode, value string) *RadixNode {
	return rt.findNode(node, value, true)
}

func (rt *RadixTree) getAllKeys(node *RadixNode) []string {
	if node == nil {
		return []string{}
	}

	var result []string

	// Collect keys from current node
	if node.isLeaf && node.keys != nil {
		result = append(result, node.keys.ToSlice()...)
	}

	// Recursively collect keys from all children
	for _, child := range node.children {
		result = append(result, rt.getAllKeys(child)...)
	}

	return result
}

// ExactSearch finds resource keys that exactly match the given value
func (rt *RadixTree) ExactSearch(indexValue string) []string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	node := rt.findExactNode(rt.root, indexValue)
	if node == nil || !node.isLeaf {
		return []string{}
	}

	return node.keys.ToSlice()
}

// GetAllIndexValues returns all index values stored in the tree
func (rt *RadixTree) GetAllIndexValues() []string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	return rt.collectIndexValues(rt.root, "")
}

// collectIndexValues recursively collects all index values from the tree
func (rt *RadixTree) collectIndexValues(node *RadixNode, currentPath string) []string {
	if node == nil {
		return []string{}
	}

	var result []string
	fullPath := currentPath + node.prefix

	// If this node is a leaf, add the full path as an index value
	if node.isLeaf {
		result = append(result, fullPath)
	}

	// Recursively collect from all children
	for _, child := range node.children {
		result = append(result, rt.collectIndexValues(child, fullPath)...)
	}

	return result
}

func (rt *RadixTree) Delete(indexValue, resourceKey string) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	return rt.delete(rt.root, indexValue, resourceKey, nil, 0)
}

func (rt *RadixTree) removeKeyFromNode(node *RadixNode, resourceKey string, parent *RadixNode, parentKey byte) bool {
	if node.keys == nil || !node.keys.Contain(resourceKey) {
		return false
	}

	node.keys.Delete(resourceKey)
	rt.size--

	if node.keys.Size() == 0 {
		node.isLeaf = false
		rt.mergeNode(node, parent, parentKey)
	}

	return true
}

func (rt *RadixTree) delete(node *RadixNode, indexValue, resourceKey string, parent *RadixNode, parentKey byte) bool {
	if node == nil {
		return false
	}

	if indexValue == "" {
		return rt.removeKeyFromNode(node, resourceKey, parent, parentKey)
	}

	if node.prefix == "" {
		child, exists := node.getChild(indexValue)
		if !exists {
			return false
		}
		return rt.delete(child, indexValue, resourceKey, node, indexValue[0])
	}

	if indexValue == node.prefix {
		return rt.removeKeyFromNode(node, resourceKey, parent, parentKey)
	}

	if strings.HasPrefix(indexValue, node.prefix) {
		remaining := indexValue[len(node.prefix):]
		child, exists := node.getChild(remaining)
		if !exists {
			return false
		}
		return rt.delete(child, remaining, resourceKey, node, remaining[0])
	}

	return false
}

func (rt *RadixTree) mergeNode(node *RadixNode, parent *RadixNode, parentKey byte) {
	if !node.isLeaf && len(node.children) == 0 {
		if parent != nil {
			delete(parent.children, parentKey)
		}
		return
	}

	if !node.isLeaf && len(node.children) == 1 {
		var onlyChild *RadixNode
		for _, child := range node.children {
			onlyChild = child
			break
		}

		node.prefix = node.prefix + onlyChild.prefix
		node.keys = onlyChild.keys
		node.children = onlyChild.children
		node.isLeaf = onlyChild.isLeaf
	}
}
