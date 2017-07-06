VERSION_VAR := main.Version
GIT_VAR := main.GitCommit
BUILD_DATE_VAR := main.BuildDate
REPO_VERSION := "0.0"
#REPO_VERSION = $$(git describe --abbrev=0 --tags)
BUILD_DATE := $$(date +%Y-%m-%d-%H:%M)
GIT_HASH := $$(git rev-parse --short HEAD)
GOBUILD_VERSION_ARGS := -ldflags "-s -X $(VERSION_VAR)=$(REPO_VERSION) -X $(GIT_VAR)=$(GIT_HASH) -X $(BUILD_DATE_VAR)=$(BUILD_DATE)"
BINARY_NAME := smith
IMAGE_NAME := atlassianlabs/smith
ARCH ?= darwin
METALINTER_CONCURRENCY ?= 4
GOVERSION := 1.8
GP := /gopath
GOPATH ?= "$$HOME/go"
MAIN_PKG := github.com/atlassian/smith/cmd/smith

setup: setup-ci
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

setup-ci:
	go get -u github.com/Masterminds/glide
	glide install --strip-vendor
	go get -u github.com/bazelbuild/rules_go/go/tools/gazelle/gazelle

update-bazel:
	gazelle -external vendored

build: fmt update-bazel
	bazel build //cmd/smith:smith

build-race: fmt update-bazel
	bazel build //cmd/smith:smith-race

build-all: fmt
	go install $$(glide nv | grep -v "/it/")
	go test -i $$(glide nv)

build-all-race: fmt
	go install -race $$(glide nv | grep -v "/it/")
	go test -i -race $$(glide nv)

fmt:
	gofmt -w=true -s $$(find . -type f -name '*.go' -not -path "./vendor/*")
	goimports -w=true -d $$(find . -type f -name '*.go' -not -path "./vendor/*")

minikube-test: fmt update-bazel
	bazel test \
		--test_output=all \
		--test_env=KUBE_PATCH_CONVERSION_DETECTOR=true \
		--test_env=KUBE_CACHE_MUTATION_DETECTOR=true \
		--test_env=KUBERNETES_SERVICE_HOST="$$(minikube ip)" \
		--test_env=KUBERNETES_SERVICE_PORT=8443 \
		--test_env=KUBERNETES_CA_PATH="$$HOME/.minikube/ca.crt" \
		--test_env=KUBERNETES_CLIENT_CERT="$$HOME/.minikube/apiserver.crt" \
		--test_env=KUBERNETES_CLIENT_KEY="$$HOME/.minikube/apiserver.key" \
		//it:go_default_test

minikube-test-sc: fmt update-bazel
	bazel test \
		--test_output=all \
		--test_env=KUBE_PATCH_CONVERSION_DETECTOR=true \
		--test_env=KUBE_CACHE_MUTATION_DETECTOR=true \
		--test_env=KUBERNETES_SERVICE_HOST="$$(minikube ip)" \
		--test_env=KUBERNETES_SERVICE_PORT=8443 \
		--test_env=KUBERNETES_CA_PATH="$$HOME/.minikube/ca.crt" \
		--test_env=KUBERNETES_CLIENT_CERT="$$HOME/.minikube/apiserver.crt" \
		--test_env=KUBERNETES_CLIENT_KEY="$$HOME/.minikube/apiserver.key" \
		--test_env=SERVICE_CATALOG_URL="http://$$(minikube ip):30080" \
		//it/svc_cat:go_default_test

minikube-run: fmt update-bazel
	KUBE_PATCH_CONVERSION_DETECTOR=true \
	KUBE_CACHE_MUTATION_DETECTOR=true \
	KUBERNETES_SERVICE_HOST="$$(minikube ip)" \
	KUBERNETES_SERVICE_PORT=8443 \
	KUBERNETES_CA_PATH="$$HOME/.minikube/ca.crt" \
	KUBERNETES_CLIENT_CERT="$$HOME/.minikube/apiserver.crt" \
	KUBERNETES_CLIENT_KEY="$$HOME/.minikube/apiserver.key" \
	bazel run //cmd/smith:smith-race

minikube-run-sc: fmt update-bazel
	KUBE_PATCH_CONVERSION_DETECTOR=true \
	KUBE_CACHE_MUTATION_DETECTOR=true \
	KUBERNETES_SERVICE_HOST="$$(minikube ip)" \
	KUBERNETES_SERVICE_PORT=8443 \
	KUBERNETES_CA_PATH="$$HOME/.minikube/ca.crt" \
	KUBERNETES_CLIENT_CERT="$$HOME/.minikube/apiserver.crt" \
	KUBERNETES_CLIENT_KEY="$$HOME/.minikube/apiserver.key" \
	bazel run //cmd/smith:smith-race -- -service-catalog-url="http://$$(minikube ip):30080"

minikube-sleeper-run: fmt update-bazel
	KUBE_PATCH_CONVERSION_DETECTOR=true \
	KUBE_CACHE_MUTATION_DETECTOR=true \
	KUBERNETES_SERVICE_HOST="$$(minikube ip)" \
	KUBERNETES_SERVICE_PORT=8443 \
	KUBERNETES_CA_PATH="$$HOME/.minikube/ca.crt" \
	KUBERNETES_CLIENT_CERT="$$HOME/.minikube/apiserver.crt" \
	KUBERNETES_CLIENT_KEY="$$HOME/.minikube/apiserver.key" \
	bazel run //examples/tprattribute/main:main-race

test: fmt update-bazel
	bazel test \
		--test_output=all \
		--test_env=KUBE_PATCH_CONVERSION_DETECTOR=true \
		--test_env=KUBE_CACHE_MUTATION_DETECTOR=true \
		//pkg/... //examples/... //cmd/...

check: build-all
	gometalinter --concurrency=$(METALINTER_CONCURRENCY) --deadline=800s ./... --vendor \
		--linter='errcheck:errcheck:-ignore=net:Close' --cyclo-over=20 \
		--disable=interfacer --disable=golint --dupl-threshold=200

check-all: build-all
	gometalinter --concurrency=$(METALINTER_CONCURRENCY) --deadline=800s ./... --vendor --cyclo-over=20 \
		--dupl-threshold=65

coveralls:
	./cover.sh
	goveralls -coverprofile=coverage.out -service=travis-ci

# Compile a static binary. Cannot be used with -race
docker: fmt update-bazel
	bazel build //cmd/smith:docker

# Compile a binary with -race. Needs to be run on a glibc-based system.
docker-race: fmt update-bazel
	bazel build //cmd/smith:docker-race

release-hash: docker
	docker push $(IMAGE_NAME):$(GIT_HASH)

release-normal: release-hash
#	docker tag $(IMAGE_NAME):$(GIT_HASH) $(IMAGE_NAME):latest
#	docker push $(IMAGE_NAME):latest
	docker tag $(IMAGE_NAME):$(GIT_HASH) $(IMAGE_NAME):$(REPO_VERSION)
	docker push $(IMAGE_NAME):$(REPO_VERSION)

release-hash-race: docker-race
	docker push $(IMAGE_NAME):$(GIT_HASH)-race

release-race: docker-race
	docker tag $(IMAGE_NAME):$(GIT_HASH)-race $(IMAGE_NAME):$(REPO_VERSION)-race
	docker push $(IMAGE_NAME):$(REPO_VERSION)-race

release: release-normal release-race

.PHONY: build
