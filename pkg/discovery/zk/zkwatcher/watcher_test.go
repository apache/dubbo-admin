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

package zkwatcher

import (
	"fmt"
	"testing"
	"time"

	"github.com/dubbogo/go-zookeeper/zk"

	"github.com/apache/dubbo-admin/pkg/core/logger"
)

func TestLocalhost(t *testing.T) {
	zkServers := []string{"localhost:2181"}
	basePath := "/services"
	conn, _, err := zk.Connect(zkServers, time.Second*10)
	if err != nil {
		logger.Fatalf("Failed to connect to zookeeper: %v", err)
	}
	watcher := NewRecursiveWatcher(conn, basePath)

	// Start listening
	if err := watcher.Start(); err != nil {
		logger.Fatalf("Failed to start watching: %v", err)
	}

	// Keep program running
	select {
	case <-watcher.stopChan:
		fmt.Println("Watcher stopped")
	}
}
