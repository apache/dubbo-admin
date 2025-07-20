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

package dubbo

var (
	golang = `
FROM golang:1.20-alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED=0 && GOPROXY=https://goproxy.cn,direct
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

WORKDIR /build

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
COPY . .
COPY ./conf /app/conf

RUN go build -ldflags="-s -w" -o /app/dubbogo ./cmd

FROM scratch

WORKDIR /app

COPY --from=builder /app/dubbogo /app/dubbogo
COPY --from=builder /app/conf /app/conf

ENV DUBBO_GO_CONFIG_PATH=/app/conf/dubbogo.yaml

ENTRYPOINT ["./dubbogo"]  
`

	java = `
FROM openjdk:8-jdk-alpine

ADD target/demo-0.0.1-SNAPSHOT.jar /app.jar

ENV JAVA_OPTS=""

ENTRYPOINT ["sh", "-c", "exec java $JAVA_OPTS -jar /app.jar"]
`

	DockerfileByRuntime = map[string]string{
		"go":   golang,
		"java": java,
	}
)
