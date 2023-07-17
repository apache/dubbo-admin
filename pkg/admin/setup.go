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

package admin

import (
	"github.com/apache/dubbo-admin/pkg/admin/router"
	core_runtime "github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/pkg/errors"
)

func Setup(rt core_runtime.Runtime) error {
	if err := RegisterDatabase(rt); err != nil {
		return errors.Wrap(err, "Database register failed")
	}
	if err := RegisterOther(rt); err != nil {
		return errors.Wrap(err, "register failed")
	}
	if err := rt.Add(router.InitRouter()); err != nil {
		return errors.Wrap(err, "Add admin bootstrap failed")
	}
	return nil
}
