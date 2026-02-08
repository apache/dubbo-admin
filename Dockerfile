# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build frontend
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/node:18-alpine AS frontend-builder

WORKDIR /ui-vue3

COPY ui-vue3/package.json ui-vue3/yarn.lock ./
RUN yarn install --frozen-lockfile
COPY ui-vue3/ ./
RUN yarn build


# Build backend
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /ui-vue3/dist app/dubbo-ui/dist


RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    go build -a -o dubbo-admin ./app/dubbo-admin/main.go


# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/alpine:3.20
RUN addgroup -g 65532 app \
 && adduser -D -u 65532 -G app app

WORKDIR /app
COPY --from=builder /app/dubbo-admin /app/dubbo-admin
USER 65532:65532

ENTRYPOINT ["./dubbo-admin"]