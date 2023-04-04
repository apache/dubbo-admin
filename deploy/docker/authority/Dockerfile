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

FROM golang:1.20.1-alpine3.17 as builder

RUN apk --update add gcc libc-dev upx ca-certificates && update-ca-certificates

ADD . /workspace

WORKDIR /workspace/cmd/authority

#RUN --mount=type=cache,target=/go \
#  go build -buildmode=pie -ldflags "-linkmode external -extldflags -static -w" \
#  -o /workspace/ca

RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go install github.com/go-delve/delve/cmd/dlv@latest

RUN --mount=type=cache,target=/go \
  go build \
  -o /workspace/cmd/authority/authority

FROM alpine:3.17

EXPOSE 30060
EXPOSE 30062

COPY --from=builder /workspace/cmd/authority/authority /authority
COPY --from=builder /go/bin/dlv /

CMD ["/authority"]
#CMD ["/dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/authority"]
