// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8s

import (
	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/logger"
	"testing"
)

func TestName(t *testing.T) {
	logger.Init()
	client := NewClient()
	client.Init(&config.Options{})
	client.VerifyServiceAccount("eyJhbGciOiJSUzI1NiIsImtpZCI6IjBxNEJqamxTVnNlWVl3R1NRR0toTVdUNm9TLVJCMGQ3MHhuYkVPM3FYUkEifQ.eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjIl0sImV4cCI6MTcwOTQzNzE1MiwiaWF0IjoxNjc3OTAxMTUyLCJpc3MiOiJodHRwczovL2t1YmVybmV0ZXMuZGVmYXVsdC5zdmMiLCJrdWJlcm5ldGVzLmlvIjp7Im5hbWVzcGFjZSI6ImFjay1vbmVwaWxvdCIsInBvZCI6eyJuYW1lIjoiYWNrLW9uZXBpbG90LWFjay1vbmVwaWxvdC01Zjc1OTVjOWQ0LWpxenFjIiwidWlkIjoiNWUzNmJiM2YtM2FhZS00NGQ3LWIwZjAtMzZhNDdiNWYxZThkIn0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJhY2stb25lcGlsb3QiLCJ1aWQiOiJkZTlmNjJlNS0yNjVmLTQ4MjctYjQ0Ni1jNTZjYjBmM2IyYWQifSwid2FybmFmdGVyIjoxNjc3OTA0NzU5fSwibmJmIjoxNjc3OTAxMTUyLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6YWNrLW9uZXBpbG90OmFjay1vbmVwaWxvdCJ9.cSWKHEq9WWWA3I-PG-l1odoM_6Vd59A7oMJRBVEYHx5ESQMT0zWacBOO03KQD5f7xmH6Lh1KE_g9t_HmhHU4oviroimExi8lTP42NsjtChffuol9MBlLMj4gJyVk251lsyqA3K8ppnl5AHWHiBjwFYKnRZMAAeuSmuKxHcIhksWPv0TC3-D3RuSPFxPIQd6yqhulwIzCp-33IVH8hJRZ7jTgMhgjOGwvet1cukx-yyIYqq3dvXWL6JhS7qtXilJhqsTg5ggaLT3_WkPOcu5-mPDikG1OogLYRwZ5gtQtNgrP5H-fHZQMiu9jlVwH4kNWvOwwa8yCl1tdNzQc_Ci3Zw")
	//client.InitController(authentication.NewHandler(), authorization.NewHandler())
	//
	//ch := make(chan struct{})
	//<-ch
}

func TestName2(t *testing.T) {

}
