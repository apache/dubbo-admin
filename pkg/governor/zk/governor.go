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

package zk

import (
	"fmt"
	"strings"

	"github.com/dubbogo/go-zookeeper/zk"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/clients"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

const zkConfigRootPath = "/dubbo/config"

type RuleGovernor struct {
	cfg         *discoverycfg.Config
	storeRouter store.Router
	emitter     events.Emitter
	conn        *zk.Conn
}

func NewZKRuleGovernor(cfg *discoverycfg.Config, router store.Router, emitter events.Emitter) (*RuleGovernor, error) {
	address := cfg.Address.ConfigCenter
	conn, err := clients.NewZKConnection(address)
	if err != nil {
		return nil, err
	}
	return &RuleGovernor{
		cfg:         cfg,
		storeRouter: router,
		emitter:     emitter,
		conn:        conn,
	}, nil
}

func (g *RuleGovernor) CreateRule(r coremodel.Resource) error {
	path := ruleConfigPath(r.ResourceMeta().Name)
	content, err := meshresource.EncodeRule(r)
	if err != nil {
		return bizerror.Wrap(err, bizerror.YamlError,
			fmt.Sprintf("failed to marshal resource spec, res: %s", r.String()))
	}
	if err := g.ensurePath(ruleConfigGroupPath()); err != nil {
		return err
	}
	_, err = g.conn.Create(path, content, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		return bizerror.Wrap(err, bizerror.ZKError,
			fmt.Sprintf("failed to create zk node, path: %s", path))
	}
	// save to store once znode is created in zk to insure local store is consistent to zk timely.
	// if save to store failed, the discovery will watch and update the store finally.
	st, err := g.storeRouter.ResourceRoute(r)
	if err != nil {
		logger.Warnf("cannot find store for rk: %s, cause: %v", r.ResourceKind(), err)
		return nil
	}
	err = st.Add(r)
	if err != nil {
		logger.Warnf("add resource to store failed, res: %s, cause: %v", r.String(), err)
		return nil
	}
	return nil
}

func (g *RuleGovernor) UpdateRule(r coremodel.Resource) error {
	path := ruleConfigPath(r.ResourceMeta().Name)
	content, err := meshresource.EncodeRule(r)
	if err != nil {
		return bizerror.Wrap(err, bizerror.YamlError,
			fmt.Sprintf("failed to marshal resource spec, res: %s", r.String()))
	}
	_, err = g.conn.Set(path, content, -1)
	if err == zk.ErrNoNode {
		if err := g.ensurePath(ruleConfigGroupPath()); err != nil {
			return err
		}
		_, err = g.conn.Create(path, content, 0, zk.WorldACL(zk.PermAll))
	}
	if err != nil {
		return bizerror.Wrap(err, bizerror.ZKError,
			fmt.Sprintf("failed to update zk node, path: %s", path))
	}
	st, err := g.storeRouter.ResourceRoute(r)
	if err != nil {
		logger.Warnf("cannot find store for rk: %s, cause: %v", r.ResourceKind(), err)
		return nil
	}
	err = st.Update(r)
	if err != nil {
		logger.Warnf("update resource in store failed, res: %s, cause: %v", r.String(), err)
	}
	return nil
}

func (g *RuleGovernor) DeleteRule(r coremodel.Resource) error {
	path := ruleConfigPath(r.ResourceMeta().Name)
	err := g.conn.Delete(path, -1)
	if err != nil {
		return bizerror.Wrap(err, bizerror.ZKError,
			fmt.Sprintf("failed to delete zk node, path: %s", path))
	}
	st, err := g.storeRouter.ResourceRoute(r)
	if err != nil {
		logger.Warnf("cannot find store for rk: %s, cause: %v", r.ResourceKind(), err)
	}
	err = st.Delete(r)
	if err != nil {
		logger.Warnf("delete resource from store failed, res: %s, cause: %v", r.String(), err)
	}
	return nil
}

func ruleConfigPath(ruleName string) string {
	return ruleConfigGroupPath() + constants.PathSeparator + ruleName
}

func ruleConfigGroupPath() string {
	return zkConfigRootPath + constants.PathSeparator + constants.RuleConfigGroup
}

func legacyRuleConfigPath(ruleName string) string {
	return zkConfigRootPath + constants.PathSeparator + ruleName
}

func (g *RuleGovernor) ensurePath(targetPath string) error {
	parts := strings.Split(strings.Trim(targetPath, constants.PathSeparator), constants.PathSeparator)
	currentPath := ""
	for _, part := range parts {
		currentPath += constants.PathSeparator + part
		exists, _, err := g.conn.Exists(currentPath)
		if err != nil {
			return bizerror.Wrap(err, bizerror.ZKError,
				fmt.Sprintf("failed to check zk node, path: %s", currentPath))
		}
		if exists {
			continue
		}
		_, err = g.conn.Create(currentPath, nil, 0, zk.WorldACL(zk.PermAll))
		if err != nil && err != zk.ErrNodeExists {
			return bizerror.Wrap(err, bizerror.ZKError,
				fmt.Sprintf("failed to create zk node, path: %s", currentPath))
		}
	}
	return nil
}
