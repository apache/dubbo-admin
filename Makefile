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


GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Git information
GIT_VERSION ?= $(shell git describe --tags --always)
GIT_COMMIT_HASH ?= $(shell git rev-parse HEAD)
GIT_TREESTATE = "clean"
GIT_DIFF = $(shell git diff --quiet >/dev/null 2>&1; if [ $$? -eq 1 ]; then echo "1"; fi)
ifeq ($(GIT_DIFF), 1)
    GIT_TREESTATE = "dirty"
endif

BUILDDATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS = "-X github.com/apache/dubbo-admin/pkg/version.gitTag=$(GIT_VERSION) \
                      -X github.com/apache/dubbo-admin/pkg/version.gitCommit=$(GIT_COMMIT_HASH) \
                      -X github.com/apache/dubbo-admin/pkg/version.gitTreeState=$(GIT_TREESTATE) \
                      -X github.com/apache/dubbo-admin/pkg/version.buildDate=$(BUILDDATE)"

# Images management
REGISTRY ?= docker.io
REGISTRY_NAMESPACE ?= apache
REGISTRY_USER_NAME?=""
REGISTRY_PASSWORD?=""

# Image URL to use all building/pushing image targets
DUBBO_ADMIN_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/dubbo-admin:${GIT_VERSION}"
DUBBO_AUTHORITY_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/dubbo-ca:${GIT_VERSION}"
DUBBO_ADMIN_UI_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/dubbo-admin-ui:${GIT_VERSION}"
DUBBO_DUBBOCTL_BUILDX_DIR ?= "./bin/dubboctl"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
SWAGGER ?= $(LOCALBIN)/swag
GOLANG_LINT ?= $(LOCALBIN)/golangci-lint
GOFUMPT  ?= $(LOCALBIN)/gofumpt


## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.10.0
SWAGGER_VERSION ?= v1.16.1
GOLANG_LINT_VERSION ?= v1.52.2
GOFUMPT_VERSION ?= latest
## docker buildx support platform
PLATFORMS ?= linux/arm64,linux/amd64

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt"  crd:allowDangerousTypes=true webhook paths="./pkg/authority/apis/..." output:crd:artifacts:config=deploy/manifests

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	#$(CONTROLLER_GEN) object:headerFile="./hack/boilerplate.go.txt"  crd:allowDangerousTypes=true paths="./..."

.PHONY: swagger
swagger: swagger-install ## Generate dubbo-admin swagger docs.
	$(SWAGGER) init --parseDependency -d cmd/admin,pkg/admin -o hack/swagger
	@rm -f hack/swagger/docs.go hack/swagger/swagger.yaml

.PHONY: fmt
fmt: gofumpt-install ## Run gofumpt against code.
	$(GOFUMPT) -l -w .

.PHONY: vet
vet: ## Run go vet against code.
	@find . -type f -name '*.go'| grep -v "/vendor/" | xargs gofmt -w -s

# Run mod tidy against code
.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: lint
lint: golangci-lint-install  ## Run golang lint against code
	GO111MODULE=on $(GOLANG_LINT) run ./... --timeout=30m -v  --disable-all --enable=gofumpt --enable=govet --enable=staticcheck --enable=ineffassign --enable=misspell

.PHONY: test
test: fmt vet  ## Run all tests.
	go test -coverprofile coverage.out -covermode=atomic ./...


.PHONY: test-dubboctl
test-dubboctl: fmt vet  ## Run tests for dubboctl
	go test -coverprofile coverage.out -covermode=atomic github.com/apache/dubbo-admin/pkg/dubboctl/...

.PHONY: test-admin
test-admin: fmt vet  ## Run tests for admin
	go test -coverprofile coverage.out -covermode=atomic github.com/apache/dubbo-admin/pkg/admin/...

.PHONY: test-authority
test-authority: fmt vet  ## Run tests for authority
	go test -coverprofile coverage.out -covermode=atomic github.com/apache/dubbo-admin/pkg/authority/...


.PHONY: echoLDFLAGS
echoLDFLAGS:
	@echo $(LDFLAGS)

.PHONY: build
build: build-admin build-authority build-dubboctl ## Build binary with the dubbo admin, authority, and dubboctl

.PHONY: all
all: generate test build

.PHONY: build-admin
build-admin:  ## Build binary with the dubbo admin.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/admin cmd/admin/main.go

.PHONY: build-authority
build-authority: ## Build binary with the dubbo authority.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/authority cmd/authority/main.go

.PHONY: build-dubboctl
build-dubboctl: ## Build binary with the dubbo dubboctl.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/dubboctl cmd/dubboctl/main.go

.PHONY: build-ui
build-ui: ## Build  the distribution of the admin ui pages.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=dubbo-admin-ui -t ${DUBBO_ADMIN_UI_IMG} ./dubbo-admin-ui -o type=local,dest=./bin/build/dubbo-admin-ui
	rm -f -R ./cmd/ui/dist/*
	rm -f ./bin/build/dubbo-admin-ui/usr/share/nginx/html/50x.html
	cp -R ./bin/build/dubbo-admin-ui/usr/share/nginx/html/* ./cmd/ui/dist/
	rm -f -R ./bin/build/dubbo-admin-ui

.PHONY: image
image: image-admin image-authority image-admin-ui ## Build docker image with the dubbo admin, authority and admin-ui

.PHONY: image-admin
image-admin: ## Build docker image with the dubbo admin.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=admin -t ${DUBBO_ADMIN_IMG} .

.PHONY: image-authority
image-authority: ## Build docker image with the dubbo authority.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=authority -t ${DUBBO_AUTHORITY_IMG} .

.PHONY: image-ui
image-ui: ## Build docker image with the dubbo admin ui.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=dubbo-admin-ui -t ${DUBBO_ADMIN_UI_IMG} ./dubbo-admin-ui



.PHONY: buildx
buildx: buildx-admin buildx-authority  ## Build and push docker cross-platform image for the dubbo admin and authority


.PHONY: buildx-admin
buildx-admin:  ## Build and push docker image with the dubbo admin for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile_admin.cross
	- docker buildx create --name project-dubbo-admin-builder
	docker buildx use project-dubbo-admin-builder
	- docker buildx build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=admin  --push --platform=$(PLATFORMS) --tag ${DUBBO_ADMIN_IMG} -f Dockerfile_admin.cross .
	#- docker buildx build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=admin  --output type=local,dest=./bin/buildx/dubbo-admin --platform=$(PLATFORMS) --tag ${DUBBO_ADMIN_IMG} -f Dockerfile_admin.cross .
	- docker buildx rm project-dubbo-admin-builder
	rm Dockerfile_admin.cross

.PHONY: buildx-authority
buildx-authority:  ## Build and push docker image with the dubbo authority for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile_authority.cross
	- docker buildx create --name project-dubbo-authority-builder
	docker buildx use project-dubbo-authority-builder
	- docker buildx build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=authority  --push --platform=$(PLATFORMS) --tag ${DUBBO_AUTHORITY_IMG} -f Dockerfile_authority.cross .
	#- docker buildx build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=authority  --output type=local,dest=./bin/buildx/dubbo-authority --platform=$(PLATFORMS) --tag ${DUBBO_AUTHORITY_IMG} -f Dockerfile_authority.cross .
	- docker buildx rm project-dubbo-authority-builder
	rm Dockerfile_authority.cross

.PHONY: buildx-dubboctl
buildx-dubboctl:  ## Build the dubboctl distribution for cross-platform support
	@rm -f -R $(DUBBO_DUBBOCTL_BUILDX_DIR)
	@mkdir $(DUBBO_DUBBOCTL_BUILDX_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(DUBBO_DUBBOCTL_BUILDX_DIR)/linux/amd64/dubboctl cmd/dubboctl/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags $(LDFLAGS) -o $(DUBBO_DUBBOCTL_BUILDX_DIR)/linux/arm64/dubboctl cmd/dubboctl/main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(DUBBO_DUBBOCTL_BUILDX_DIR)/darwin/amd64/dubboctl cmd/dubboctl/main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags $(LDFLAGS) -o $(DUBBO_DUBBOCTL_BUILDX_DIR)/darwin/arm64/dubboctl cmd/dubboctl/main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(DUBBO_DUBBOCTL_BUILDX_DIR)/windows/amd64/dubboctl.exe cmd/dubboctl/main.go

	tar -cvzf $(DUBBO_DUBBOCTL_BUILDX_DIR)/dubboctl-${GIT_VERSION}-linux-amd64.tar.gz  -C $(DUBBO_DUBBOCTL_BUILDX_DIR)/linux/amd64/ dubboctl
	tar -cvzf $(DUBBO_DUBBOCTL_BUILDX_DIR)/dubboctl-${GIT_VERSION}-linux-arm64.tar.gz  -C $(DUBBO_DUBBOCTL_BUILDX_DIR)/linux/arm64/ dubboctl

	tar -cvzf $(DUBBO_DUBBOCTL_BUILDX_DIR)/dubboctl-${GIT_VERSION}-osx-arm64.tar.gz  -C $(DUBBO_DUBBOCTL_BUILDX_DIR)/darwin/arm64/ dubboctl
	tar -cvzf $(DUBBO_DUBBOCTL_BUILDX_DIR)/dubboctl-${GIT_VERSION}-osx.tar.gz  -C $(DUBBO_DUBBOCTL_BUILDX_DIR)/darwin/amd64/ dubboctl
	zip  $(DUBBO_DUBBOCTL_BUILDX_DIR)/dubboctl-${GIT_VERSION}-win.zip -D -j $(DUBBO_DUBBOCTL_BUILDX_DIR)/windows/amd64/dubboctl.exe



.PHONY: push-images
push-images: push-image-admin push-image-ui push-image-authority

.PHONY: push-image-admin
push-image-admin: ## Push admin images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${DUBBO_ADMIN_IMG}

.PHONY: push-image-authority
push-image-authority: ## Push authority images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${DUBBO_AUTHORITY_IMG}

.PHONY: push-image-ui
push-image-ui: ## Push admin ui images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${DUBBO_ADMIN_UI_IMG}




KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(LOCALBIN) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(LOCALBIN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: swagger-install
swagger-install: $(LOCALBIN) ## Download swagger locally if necessary.
	test -s $(LOCALBIN)/swag  || \
	GOBIN=$(LOCALBIN) go install  github.com/swaggo/swag/cmd/swag@$(SWAGGER_VERSION)


GOLANG_LINT_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
.PHONY: golangci-lint-install
golangci-lint-install: $(LOCALBIN) ## Download golangci lint locally if necessary.
	test -s $(LOCALBIN)/golangci-lint  && $(LOCALBIN)/golangci-lint --version | grep -q $(GOLANG_LINT_VERSION) || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANG_LINT_VERSION)


.PHONY: gofumpt-install
gofumpt-install: $(LOCALBIN) ## Download gofumpt locally if necessary.
	test -s $(LOCALBIN)/gofumpt || \
	GOBIN=$(LOCALBIN) go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
