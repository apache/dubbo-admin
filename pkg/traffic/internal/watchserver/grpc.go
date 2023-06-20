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

package watchserver

import (
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/apache/dubbo-admin/pkg/traffic/internal/pb"
)

const NetWork = "tcp"

type WatchGrpcConfig struct {
	Addr         string
	RegisterFunc func(server *grpc.Server)
}

func RegisterGrpc() *grpc.Server {
	c := WatchGrpcConfig{
		// TODO the Addr config
		Addr: "127.0.0.1:8000",
		RegisterFunc: func(g *grpc.Server) {
			pb.RegisterWatchServer(g, WatchSrv)
		},
	}
	s := grpc.NewServer()
	c.RegisterFunc(s)
	lis, err := net.Listen(NetWork, c.Addr)
	if err != nil {
		log.Println("cannot listen")
	}
	go func() {
		err = s.Serve(lis)
		if err != nil {
			log.Println("server started error", err)
			return
		}
	}()
	return s
}
