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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	dubbo "dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/server"
	hessian "github.com/apache/dubbo-go-hessian2"
)

// Explicit int32 fields make the nested integer declaration part of the wire contract.
type Item struct {
	Value int32 `json:"value"`
}

func (*Item) JavaClassName() string { return "example.Item" }

type Envelope struct {
	Item *Item `json:"item"`
}

func (*Envelope) JavaClassName() string { return "example.Envelope" }

type EchoService struct{ requests atomic.Int64 }

func (s *EchoService) Echo(ctx context.Context, value string) (string, error) {
	s.requests.Add(1)
	return value, nil
}
func (s *EchoService) EchoInt(ctx context.Context, value int32) (int32, error) {
	s.requests.Add(1)
	return value, nil
}
func (s *EchoService) EchoNested(ctx context.Context, value *Envelope) (*Envelope, error) {
	s.requests.Add(1)
	return value, nil
}
func (s *EchoService) NoArgs(ctx context.Context) (string, error) {
	s.requests.Add(1)
	return "no-args", nil
}
func (s *EchoService) EchoArray(ctx context.Context, value []int32) ([]int32, error) {
	s.requests.Add(1)
	return value, nil
}
func (s *EchoService) Attachments(ctx context.Context) (string, error) {
	s.requests.Add(1)
	attachments := ctx.Value(constant.AttachmentKey)
	// Triple HTTP/2 normalizes header names and may provide multiple values.
	for _, key := range []string{"traceId", "traceid"} {
		var value any
		switch a := attachments.(type) {
		case map[string]any:
			value = a[key]
		case map[string]string:
			value = a[key]
		case map[string][]string:
			value = a[key]
		}
		switch v := value.(type) {
		case string:
			if v != "" {
				return v, nil
			}
		case []string:
			if len(v) > 0 && v[0] != "" {
				return v[0], nil
			}
		}
	}
	return "", nil
}
func (s *EchoService) Delay(ctx context.Context, delayMs int32) (string, error) {
	s.requests.Add(1)
	if delayMs < 0 || delayMs > 10000 {
		return "", fmt.Errorf("delayMs must be between 0 and 10000")
	}
	timer := time.NewTimer(time.Duration(delayMs) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
		return "delayed", nil
	}
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func port(key, fallback string) int {
	v, e := strconv.Atoi(env(key, fallback))
	if e != nil || v < 1 || v > 65535 {
		log.Fatalf("invalid %s", key)
	}
	return v
}
func main() {
	hessian.RegisterPOJO(&Item{})
	hessian.RegisterPOJO(&Envelope{})
	host := env("PROVIDER_HOST", "127.0.0.1")
	ins, err := dubbo.NewInstance(dubbo.WithName("generic-go-v330"), dubbo.WithRemoteMetadata(),
		dubbo.WithRegistry(registry.WithZookeeper(), registry.WithAddress(env("ZOOKEEPER_ADDR", "127.0.0.1:2181")), registry.WithRegisterServiceAndInterface()),
		dubbo.WithProtocol(protocol.WithID("dubbo"), protocol.WithDubbo(), protocol.WithIp(host), protocol.WithPort(port("DUBBO_PORT", "20880"))),
		dubbo.WithProtocol(protocol.WithID("tri"), protocol.WithTriple(), protocol.WithIp(host), protocol.WithPort(port("TRIPLE_PORT", "50051"))))
	if err != nil {
		log.Fatal(err)
	}
	srv, err := ins.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	echo := &EchoService{}
	// Pass a non-nil ServiceInfo to avoid dubbo-go v3.3.0 typed-nil panic in Server.exportServices.
	if err = srv.Register(echo, &common.ServiceInfo{InterfaceName: "example.GoEchoService"}, server.WithInterface("example.GoEchoService"), server.WithProtocolIDs([]string{"dubbo", "tri"})); err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int64{"count": echo.requests.Load()})
	})
	go func() {
		log.Fatal((&http.Server{Addr: env("OBSERVATION_ADDR", "127.0.0.1:18090"), Handler: mux, ReadHeaderTimeout: 3 * time.Second}).ListenAndServe())
	}()
	if err = srv.Serve(); err != nil {
		log.Fatal(err)
	}
}
