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

# Build the image binary
FROM golang:1.21-alpine as builder


# Build argments
ARG TARGETOS
ARG TARGETARCH
ARG LDFLAGS
ARG CN_MIRROR

WORKDIR /go/src/github.com/apache/dubbo-kubernetes

# Copy the go source
COPY . .

# disable cgo
ENV CGO_ENABLED=0

#RUN if [[ "${PKGNAME}" == "authority" ]]; then apk --update add gcc libc-dev upx ca-certificates && update-ca-certificates; fi

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN if [[ -n "${CN_MIRROR}" ]]; then go env -w GOPROXY=https://goproxy.cn,direct; fi
RUN go env
RUN go mod download

# Build
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -ldflags="${LDFLAGS}" -a -o dubbo-cp ./app/dubbo-cp/main.go

# ==============
# Final image
# Using a minimal operating system image
# instead of a scratch image to allow for debugging
FROM ubuntu:jammy

# Build
WORKDIR /app/
ARG PKGNAME
COPY --from=builder /go/src/github.com/apache/dubbo-kubernetes/dubbo-cp .

ENTRYPOINT ["./dubbo-cp", "run"]
