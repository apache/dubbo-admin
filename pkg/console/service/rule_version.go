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

package service

import (
	"context"
	"fmt"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
)

type RuleMutationOptions struct {
	ExpectedVersionID *int64
	Author            string
}

func ruleVersioning(ctx consolectx.Context) versioning.Service {
	if ctx == nil {
		return nil
	}
	return ctx.RuleVersioning()
}

func checkExpectedVersion(ctx consolectx.Context, kindName RuleKindName, opts RuleMutationOptions) error {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil
	}
	return svc.CheckExpected(kindName.Kind, kindName.Mesh, kindName.Name, opts.ExpectedVersionID)
}

func putAdminHint(ctx consolectx.Context, res coremodel.Resource, op versioning.Operation, opts RuleMutationOptions) error {
	svc := ruleVersioning(ctx)
	if svc == nil {
		return nil
	}
	return svc.PutAdminHint(res, op, versioning.SourceAdmin, opts.Author, "", nil)
}

type RuleKindName struct {
	Kind coremodel.ResourceKind
	Mesh string
	Name string
}

func getExistingRule(ctx consolectx.Context, kindName RuleKindName) (coremodel.Resource, error) {
	key := coremodel.BuildResourceKey(kindName.Mesh, kindName.Name)
	res, exists, err := ctx.ResourceManager().GetByKey(kindName.Kind, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("%s %s does not exist", kindName.Kind, key)
	}
	return res, nil
}

func ListRuleVersions(ctx consolectx.Context, kindName RuleKindName) (*versioning.ListResult, error) {
	return ctx.RuleVersioning().List(kindName.Kind, kindName.Mesh, kindName.Name)
}

func GetRuleVersion(ctx consolectx.Context, kindName RuleKindName, versionID int64) (*versioning.Version, error) {
	return ctx.RuleVersioning().Get(kindName.Kind, kindName.Mesh, kindName.Name, versionID)
}

func DiffRuleVersion(ctx consolectx.Context, kindName RuleKindName, versionID int64, against string) (*versioning.DiffResult, error) {
	return ctx.RuleVersioning().Diff(kindName.Kind, kindName.Mesh, kindName.Name, versionID, against)
}

func RollbackRuleVersion(reqCtx context.Context, ctx consolectx.Context, kindName RuleKindName, versionID int64, reason string, expected *int64, author string) (*versioning.Version, error) {
	return ctx.RuleVersioning().Rollback(reqCtx, ctx.ResourceManager(), kindName.Kind, kindName.Mesh, kindName.Name, versionID, reason, expected, author)
}
