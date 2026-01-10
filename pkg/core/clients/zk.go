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

package clients

import (
	"fmt"
	"net/url"
	"time"

	"github.com/dubbogo/go-zookeeper/zk"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/logger"
)

type ZKLogger struct{}

func (z *ZKLogger) Printf(s string, i ...interface{}) {
	logger.Debugf(s, i...)
}

// NewZKConnection creates a new zookeeper connection
// address format: zookeeper://host:port
func NewZKConnection(address string) (*zk.Conn, error) {
	zkUrl, err := url.Parse(address)
	if err != nil {
		return nil, bizerror.Wrap(err, bizerror.ZKError,
			fmt.Sprintf("cannot parse url for zookeeper, url: %s", address))
	}
	conn, _, err := zk.Connect([]string{zkUrl.Host}, time.Second*1, func(c *zk.Conn) {
		c.SetLogger(&ZKLogger{})
	})
	if err != nil {
		logger.Errorf("cannot connect to zookeeper, url: %s", address)
		return nil, bizerror.Wrap(err, bizerror.ZKError,
			fmt.Sprintf("cannot connect to zookeeper, url: %s", address))
	}
	return conn, nil
}
