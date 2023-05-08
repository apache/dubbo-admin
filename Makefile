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

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.10.0
SWAGGER_VERSION ?= v1.16.1

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

.PHONY: dubbo-admin-swagger-gen
dubbo-admin-swagger-gen: swagger-install ## Generate dubbo-admin swagger docs.
	$(SWAGGER) init -d cmd/admin,pkg/admin -o hack/swagger/docs

.PHONY: dubbo-admin-swagger-ui
dubbo-admin-swagger-ui: dubbo-admin-swagger-gen ## Generate dubbo-admin swagger docs and start swagger ui.
	@echo "access swagger url: http://127.0.0.1:38081/swagger/index.html"
	cd hack/swagger; go run main.go

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	@find . -type f -name '*.go'| grep -v "/vendor/" | xargs gofmt -w -s

# Run mod tidy against code
.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: lint
lint: golangci-lint  ## Run golang lint against code
	@$(GOLANG_LINT) run ./...

.PHONY: test
test: fmt vet  ## Run tests.
	go test -coverprofile coverage.out -covermode=atomic ./...

.PHONY: echoLDFLAGS
echoLDFLAGS:
	@echo $(LDFLAGS)


.PHONY: build
build: dubbo-admin dubbo-authority

.PHONY: all
all: generate test dubbo-admin dubbo-authority

.PHONY: dubbo-admin
dubbo-admin: ## Build binary with the dubbo admin.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/admi cmd/admin/main.go

.PHONY: dubbo-authority
dubbo-authority: ## Build binary with the dubbo authority.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/authority cmd/authority/main.go

.PHONY: images
images: image-dubbo-admin image-dubbo-authority  image-dubbo-admin-ui

.PHONY: image-dubbo-admin
image-dubbo-admin: ## Build docker image with the dubbo admin.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=admin -t ${DUBBO_ADMIN_IMG} .



.PHONY: image-dubbo-admin-buildx
image-dubbo-admin-buildx:  ## Build and push docker image for the dubbo admin for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile_admin.cross
	- docker buildx create --name project-dubbo-admin-builder
	docker buildx use project-dubbo-admin-builder
	- docker buildx build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=admin  --push --platform=$(PLATFORMS) --tag ${DUBBO_ADMIN_IMG} -f Dockerfile_admin.cross .
	#- docker buildx build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=admin  --output type=local,dest=./bin/buildx/dubbo-admin --platform=$(PLATFORMS) --tag ${DUBBO_ADMIN_IMG} -f Dockerfile_admin.cross .
	- docker buildx rm project-dubbo-admin-builder
	rm Dockerfile_admin.cross

.PHONY: image-dubbo-authority
image-dubbo-authority: ## Build docker image with the dubbo authority.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=authority -t ${DUBBO_AUTHORITY_IMG} .

.PHONY: image-dubbo-authority-buildx
image-dubbo-authority-buildx:  ## Build and push docker image for the dubbo authority for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile_authority.cross
	- docker buildx create --name project-dubbo-authority-builder
	docker buildx use project-dubbo-authority-builder
	- docker buildx build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=authority  --push --platform=$(PLATFORMS) --tag ${DUBBO_AUTHORITY_IMG} -f Dockerfile_authority.cross .
	#- docker buildx build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=authority  --output type=local,dest=./bin/buildx/dubbo-authority --platform=$(PLATFORMS) --tag ${DUBBO_AUTHORITY_IMG} -f Dockerfile_authority.cross .
	- docker buildx rm project-dubbo-authority-builder
	rm Dockerfile_authority.cross

.PHONY: image-dubbo-admin-ui
image-dubbo-admin-ui: ## Build docker image with the dubbo-admin-ui.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=dubbo-admin-ui -t ${DUBBO_ADMIN_UI_IMG} ./dubbo-admin-ui


.PHONY: push-images
push-images: push-image-admin push-image-admin-ui push-image-authority

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

.PHONY: push-image-admin-ui
push-image-admin-ui: ## Push admin ui images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${DUBBO_ADMIN_UI_IMG}



KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: swagger-install
swagger-install: $(SWAGGER) ## Download swagger locally if necessary.
$(SWAGGER): $(LOCALBIN)
	test -s $(LOCALBIN)/swag  || \
	GOBIN=$(LOCALBIN) go install  github.com/swaggo/swag/cmd/swag@$(SWAGGER_VERSION)