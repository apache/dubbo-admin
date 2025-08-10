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

package store

import (
	"fmt"
	"strings"
	"time"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

const (
	PathLabel = "dubbo.io/path"
)

type CreateOptions struct {
	Name         string
	Mesh         string
	CreationTime time.Time
	Owner        coremodel.Resource
	Labels       map[string]string
}

type CreateOptionsFunc func(*CreateOptions)

func NewCreateOptions(fs ...CreateOptionsFunc) *CreateOptions {
	opts := &CreateOptions{
		Labels: map[string]string{},
	}
	for _, f := range fs {
		f(opts)
	}
	return opts
}

func CreateByApplication(app string) CreateOptionsFunc {
	return func(opts *CreateOptions) {
		opts.Labels[meshproto.ApplicationLabel] = app
	}
}

func CreateByService(service string) CreateOptionsFunc {
	return func(opts *CreateOptions) {
		opts.Labels[meshproto.ServiceLabel] = service
	}
}

func CreateByID(id string) CreateOptionsFunc {
	return func(opts *CreateOptions) {
		opts.Labels[meshproto.IDLabel] = id
	}
}

func CreateByServiceVersion(serviceVersion string) CreateOptionsFunc {
	return func(opts *CreateOptions) {
		opts.Labels[meshproto.ServiceVersionLabel] = serviceVersion
	}
}

func CreateByServiceGroup(serviceGroup string) CreateOptionsFunc {
	return func(opts *CreateOptions) {
		opts.Labels[meshproto.ServiceGroupLabel] = serviceGroup
	}
}

func CreateByPath(path string) CreateOptionsFunc {
	return func(opts *CreateOptions) {
		opts.Labels[PathLabel] = path
	}
}

func CreateByKey(name, mesh string) CreateOptionsFunc {
	return func(opts *CreateOptions) {
		opts.Name = name
		opts.Mesh = mesh
	}
}

func CreatedAt(creationTime time.Time) CreateOptionsFunc {
	return func(opts *CreateOptions) {
		opts.CreationTime = creationTime
	}
}

func CreateWithOwner(owner coremodel.Resource) CreateOptionsFunc {
	return func(opts *CreateOptions) {
		opts.Owner = owner
	}
}

func CreateWithLabels(labels map[string]string) CreateOptionsFunc {
	return func(opts *CreateOptions) {
		opts.Labels = labels
	}
}

type UpdateOptions struct {
	Name             string
	Mesh             string
	ModificationTime time.Time
	Labels           map[string]string
}

func ModifiedAt(modificationTime time.Time) UpdateOptionsFunc {
	return func(opts *UpdateOptions) {
		opts.ModificationTime = modificationTime
	}
}

func UpdateByApplication(app string) UpdateOptionsFunc {
	return func(opts *UpdateOptions) {
		opts.Labels[meshproto.ApplicationLabel] = app
	}
}

func UpdateByService(service string) UpdateOptionsFunc {
	return func(opts *UpdateOptions) {
		opts.Labels[meshproto.ServiceLabel] = service
	}
}

func UpdateByID(id string) UpdateOptionsFunc {
	return func(opts *UpdateOptions) {
		opts.Labels[meshproto.IDLabel] = id
	}
}

func UpdateByServiceVersion(serviceVersion string) UpdateOptionsFunc {
	return func(opts *UpdateOptions) {
		opts.Labels[meshproto.ServiceVersionLabel] = serviceVersion
	}
}

func UpdateByServiceGroup(serviceGroup string) UpdateOptionsFunc {
	return func(opts *UpdateOptions) {
		opts.Labels[meshproto.ServiceGroupLabel] = serviceGroup
	}
}

func UpdateByKey(name, mesh string) UpdateOptionsFunc {
	return func(opts *UpdateOptions) {
		opts.Name = name
		opts.Mesh = mesh
	}
}

func UpdateWithPath(path string) UpdateOptionsFunc {
	return func(opts *UpdateOptions) {
		opts.Labels[PathLabel] = path
	}
}

func UpdateWithLabels(labels map[string]string) UpdateOptionsFunc {
	return func(opts *UpdateOptions) {
		opts.Labels = labels
	}
}

type UpdateOptionsFunc func(*UpdateOptions)

func NewUpdateOptions(fs ...UpdateOptionsFunc) *UpdateOptions {
	opts := &UpdateOptions{
		Labels: map[string]string{},
	}
	for _, f := range fs {
		f(opts)
	}
	return opts
}

type DeleteOptions struct {
	Name   string
	Mesh   string
	Labels map[string]string
}

type DeleteOptionsFunc func(*DeleteOptions)

func NewDeleteOptions(fs ...DeleteOptionsFunc) *DeleteOptions {
	opts := &DeleteOptions{
		Labels: map[string]string{},
	}
	for _, f := range fs {
		f(opts)
	}
	return opts
}

func DeleteByPath(path string) DeleteOptionsFunc {
	return func(opts *DeleteOptions) {
		opts.Labels[PathLabel] = path
	}
}

func DeleteByApplication(app string) DeleteOptionsFunc {
	return func(opts *DeleteOptions) {
		opts.Labels[meshproto.ApplicationLabel] = app
	}
}

func DeleteByService(service string) DeleteOptionsFunc {
	return func(opts *DeleteOptions) {
		opts.Labels[meshproto.ServiceLabel] = service
	}
}

func DeleteByID(id string) DeleteOptionsFunc {
	return func(opts *DeleteOptions) {
		opts.Labels[meshproto.IDLabel] = id
	}
}

func DeleteByServiceVersion(serviceVersion string) DeleteOptionsFunc {
	return func(opts *DeleteOptions) {
		opts.Labels[meshproto.ServiceVersionLabel] = serviceVersion
	}
}

func DeleteByServiceGroup(serviceGroup string) DeleteOptionsFunc {
	return func(opts *DeleteOptions) {
		opts.Labels[meshproto.ServiceGroupLabel] = serviceGroup
	}
}

func DeleteByKey(name, mesh string) DeleteOptionsFunc {
	return func(opts *DeleteOptions) {
		opts.Name = name
		opts.Mesh = mesh
	}
}

type DeleteAllOptions struct {
	Mesh string
}

type DeleteAllOptionsFunc func(*DeleteAllOptions)

func DeleteAllByMesh(mesh string) DeleteAllOptionsFunc {
	return func(opts *DeleteAllOptions) {
		opts.Mesh = mesh
	}
}

func NewDeleteAllOptions(fs ...DeleteAllOptionsFunc) *DeleteAllOptions {
	opts := &DeleteAllOptions{}
	for _, f := range fs {
		f(opts)
	}
	return opts
}

type GetOptions struct {
	Name       string
	Mesh       string
	Version    string
	Type       string
	Consistent bool
	Labels     map[string]string
}

type GetOptionsFunc func(*GetOptions)

func NewGetOptions(fs ...GetOptionsFunc) *GetOptions {
	opts := &GetOptions{
		Labels: map[string]string{},
	}
	for _, f := range fs {
		f(opts)
	}
	return opts
}

func (g *GetOptions) HashCode() string {
	return fmt.Sprintf("%s:%s", g.Name, g.Mesh)
}

func GetByPath(path string) GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Labels[PathLabel] = path
	}
}

func GetByRevision(revision string) GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Labels[meshproto.RevisionLabel] = revision
	}
}

func GetByType(t string) GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Type = t
	}
}

func GetByApplication(app string) GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Labels[meshproto.ApplicationLabel] = app
	}
}

func GetByService(service string) GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Labels[meshproto.ServiceLabel] = service
	}
}

func GetByID(id string) GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Labels[meshproto.IDLabel] = id
	}
}

func GetByServiceVersion(serviceVersion string) GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Labels[meshproto.ServiceVersionLabel] = serviceVersion
	}
}

func GetByServiceGroup(serviceGroup string) GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Labels[meshproto.ServiceGroupLabel] = serviceGroup
	}
}
func GetByKey(name, mesh string) GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Name = name
		opts.Mesh = mesh
	}
}

func GetByVersion(version string) GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Version = version
	}
}

// GetConsistent forces consistency if storage provides eventual consistency like read replica for Postgres.
func GetConsistent() GetOptionsFunc {
	return func(opts *GetOptions) {
		opts.Consistent = true
	}
}

func (l *GetOptions) Predicate(r coremodel.Resource) bool {
	if l.Mesh != "" && r.Mesh() != l.Mesh {
		return false
	}

	if l.Version != "" && r.Meta().ResourceVersion != l.Version {
		return false
	}

	if len(l.Labels) > 0 {
		for k, v := range l.Labels {
			if r.Meta().Labels[k] != v {
				return false
			}
		}
	}

	return true
}

type (
	ListFilterFunc func(rs coremodel.Resource) bool
)

type ListOptions struct {
	Mesh                string
	Labels              map[string]string
	PageSize            int
	PageOffset          string
	FilterFunc          ListFilterFunc
	NameContains        string
	NameEquals          string
	ApplicationContains string
	Ordered             bool
	ResourceKeys        map[string]struct{}
}

type ListOptionsFunc func(*ListOptions)

func NewListOptions(fs ...ListOptionsFunc) *ListOptions {
	opts := &ListOptions{
		Labels:       map[string]string{},
		ResourceKeys: map[string]struct{}{},
	}
	for _, f := range fs {
		f(opts)
	}
	return opts
}

// Filter returns true if the item passes the filtering criteria
func (l *ListOptions) Filter(rs coremodel.Resource) bool {
	if l.FilterFunc == nil {
		return true
	}

	return l.FilterFunc(rs)
}

func ListByPath(path string) ListOptionsFunc {
	return func(opts *ListOptions) {
		opts.Labels[PathLabel] = path
	}
}

func ListByApplicationContains(app string) ListOptionsFunc {
	return func(opts *ListOptions) {
		opts.ApplicationContains = app
	}
}

func ListByApplication(app string) ListOptionsFunc {
	return func(opts *ListOptions) {
		opts.Labels[meshproto.ApplicationLabel] = app
	}
}

func ListByNameContains(name string) ListOptionsFunc {
	return func(opts *ListOptions) {
		opts.NameContains = name
	}
}

func ListByNameEquals(name string) ListOptionsFunc {
	return func(opts *ListOptions) {
		opts.NameEquals = name
	}
}

func ListByMesh(mesh string) ListOptionsFunc {
	return func(opts *ListOptions) {
		opts.Mesh = mesh
	}
}

func ListByPage(size int, offset string) ListOptionsFunc {
	return func(opts *ListOptions) {
		opts.PageSize = size
		opts.PageOffset = offset
	}
}

func ListByFilterFunc(filterFunc ListFilterFunc) ListOptionsFunc {
	return func(opts *ListOptions) {
		opts.FilterFunc = filterFunc
	}
}

func ListOrdered() ListOptionsFunc {
	return func(opts *ListOptions) {
		opts.Ordered = true
	}
}

func (l *ListOptions) IsCacheable() bool {
	return l.FilterFunc == nil
}

func (l *ListOptions) HashCode() string {
	return fmt.Sprintf("%s:%t:%s:%d:%s", l.Mesh, l.Ordered, l.NameContains, l.PageSize, l.PageOffset)
}

func (l *ListOptions) Predicate(r coremodel.Resource) bool {
	if l.Mesh != "" && r.Mesh() != l.Mesh {
		return false
	}
	if l.NameEquals != "" && r.Meta().Name != l.NameEquals {
		return false
	}

	if l.NameContains != "" && !strings.Contains(r.Meta().Name, l.NameContains) {
		return false
	}

	if l.ApplicationContains != "" && !strings.Contains(r.Meta().Labels[meshproto.ApplicationLabel], l.ApplicationContains) {
		return false
	}

	if len(l.Labels) > 0 {
		for k, v := range l.Labels {
			if r.Meta().Labels[k] != v {
				return false
			}
		}
	}

	return l.Filter(r)
}
