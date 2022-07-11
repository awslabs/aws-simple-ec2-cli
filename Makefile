MAKEFILE_PATH = $(dir $(realpath -s $(firstword $(MAKEFILE_LIST))))
PROJECT_IMPORT_DIR = simple-ec2
BUILD_DIR_PATH = ${MAKEFILE_PATH}/build
CLI_BINARY_NAME = simple-ec2
VERSION ?= $(shell git describe --tags --always --dirty)
IMG ?= amazon/aws-simple-ec2-cli
BIN ?= simple-ec2
REPO_SHORT_NAME ?= aws-simple-ec2-cli
REPO_FULL_NAME ?= awslabs/${REPO_SHORT_NAME}
GOOS ?= $(uname | tr '[:upper:]' '[:lower:]')
GOARCH ?= amd64
GOPROXY ?= "https://proxy.golang.org,direct"
SUPPORTED_PLATFORMS ?= "windows/amd64,darwin/amd64,linux/amd64,linux/arm64,linux/arm"
SELECTOR_PKG_VERSION_VAR=github.com/awslabs/aws-simple-ec2-cli/v2/pkg/selector.versionID
LATEST_RELEASE_TAG=$(shell git describe --tags --abbrev=0)
PREVIOUS_RELEASE_TAG=$(shell git describe --abbrev=0 --tags `git rev-list --tags --skip=1  --max-count=1`)

# The main CloudFormation template for creating a new stack during launch
SIMPLE_EC2_CLOUDFORMATION_TEMPLATE_FILE=${MAKEFILE_PATH}/cloudformation_template.json
SIMPLE_EC2_CLOUDFORMATION_TEMPLATE_ENCODED=$(shell cat ${SIMPLE_EC2_CLOUDFORMATION_TEMPLATE_FILE} | base64 | tr -d \\n)
SIMPLE_EC2_CLOUDFORMATION_TEMPLATE_VAR=${PROJECT_IMPORT_DIR}/pkg/cfn.SimpleEc2CloudformationTemplateEncoded

# The CloudFormation template for e2e cfn test
E2E_CFN_TEST_CLOUDFORMATION_TEMPLATE_FILE=${MAKEFILE_PATH}/test/e2e/e2e-cfn-test/cloudformation_template.json
E2E_CFN_TEST_CLOUDFORMATION_TEMPLATE_ENCODED=$(shell cat ${E2E_CFN_TEST_CLOUDFORMATION_TEMPLATE_FILE} | base64 | tr -d \\n)
E2E_CFN_TEST_CLOUDFORMATION_TEMPLATE_VAR=${PROJECT_IMPORT_DIR}/pkg/cfn.E2eCfnTestCloudformationTemplateEncoded

# The CloudFormation template for e2e connect test
E2E_CONNECT_TEST_CLOUDFORMATION_TEMPLATE_FILE=${MAKEFILE_PATH}/test/e2e/e2e-connect-test/cloudformation_template.json
E2E_CONNECT_TEST_CLOUDFORMATION_TEMPLATE_ENCODED=$(shell cat ${E2E_CONNECT_TEST_CLOUDFORMATION_TEMPLATE_FILE} | base64 | tr -d \\n)
E2E_CONNECT_TEST_CLOUDFORMATION_TEMPLATE_VAR=${PROJECT_IMPORT_DIR}/pkg/cfn.E2eConnectTestCloudformationTemplateEncoded

# The CloudFormation template for e2e ec2helper test
E2E_EC2HELPER_TEST_CLOUDFORMATION_TEMPLATE_FILE=${MAKEFILE_PATH}/test/e2e/e2e-ec2helper-test/cloudformation_template.json
E2E_EC2HELPER_TEST_CLOUDFORMATION_TEMPLATE_ENCODED=$(shell cat ${E2E_EC2HELPER_TEST_CLOUDFORMATION_TEMPLATE_FILE} | base64 | tr -d \\n)
E2E_EC2HELPER_TEST_CLOUDFORMATION_TEMPLATE_VAR=${PROJECT_IMPORT_DIR}/pkg/cfn.E2eEc2helperTestCloudformationTemplateEncoded
BUILD_VERSION_VAR=${PROJECT_IMPORT_DIR}/pkg/version.BuildInfo

EMBED_FLAGS=-ldflags '-X "${SIMPLE_EC2_CLOUDFORMATION_TEMPLATE_VAR}=${SIMPLE_EC2_CLOUDFORMATION_TEMPLATE_ENCODED}"\
 -X "${E2E_CFN_TEST_CLOUDFORMATION_TEMPLATE_VAR}=${E2E_CFN_TEST_CLOUDFORMATION_TEMPLATE_ENCODED}"\
 -X "${E2E_CONNECT_TEST_CLOUDFORMATION_TEMPLATE_VAR}=${E2E_CONNECT_TEST_CLOUDFORMATION_TEMPLATE_ENCODED}"\
 -X "${E2E_EC2HELPER_TEST_CLOUDFORMATION_TEMPLATE_VAR}=${E2E_EC2HELPER_TEST_CLOUDFORMATION_TEMPLATE_ENCODED}"\
 -X "${BUILD_VERSION_VAR}=${VERSION}"'

E2E_TEST_PACKAGES=simple-ec2/test/e2e/...

GO_TEST=go test ${EMBED_FLAGS} -bench=. ${MAKEFILE_PATH}
DELETE_STACK=aws cloudformation delete-stack --stack-name 

$(shell mkdir -p ${BUILD_DIR_PATH} && touch ${BUILD_DIR_PATH}/_go.mod)

version:
	@echo ${VERSION}

repo-full-name:
	@echo ${REPO_FULL_NAME}

bin-name:
	@echo ${BIN}

latest-release-tag:
	@echo ${LATEST_RELEASE_TAG}

previous-release-tag:
	@echo ${PREVIOUS_RELEASE_TAG}

clean:
	rm -rf ${BUILD_DIR_PATH}/ && go clean -testcache ./...

compile:
	go build ${EMBED_FLAGS} -o ${BUILD_DIR_PATH}/${CLI_BINARY_NAME} ${MAKEFILE_PATH}/main.go

build: clean compile
	
build-binaries:
	${MAKEFILE_PATH}/scripts/build-binaries -p ${SUPPORTED_PLATFORMS} -v ${VERSION}

build-docker-images:
	${MAKEFILE_PATH}/scripts/build-docker-images -p ${SUPPORTED_PLATFORMS} -r ${IMG} -v ${VERSION}

unit-test:
	${GO_TEST}/pkg/... -v -coverprofile=coverage.out -covermode=atomic -outputdir=${BUILD_DIR_PATH}
	go tool cover -func ${BUILD_DIR_PATH}/coverage.out

e2e-test:
	${GO_TEST}/test/e2e/... -v
	${DELETE_STACK}simple-ec2-e2e-cfn-test
	${DELETE_STACK}simple-ec2-e2e-connect-test
	${DELETE_STACK}simple-ec2-e2e-ec2helper-test

e2e-cfn-test:
	${GO_TEST}/test/e2e/e2e-cfn-test/... -v
	${DELETE_STACK}simple-ec2-e2e-cfn-test

e2e-connect-test:
	${GO_TEST}/test/e2e/e2e-connect-test/... -v
	${DELETE_STACK}simple-ec2-e2e-connect-test

e2e-ec2helper-test:
	${GO_TEST}/test/e2e/e2e-ec2helper-test/... -v
	${DELETE_STACK}simple-ec2-e2e-ec2helper-test

license-test:
	${MAKEFILE_PATH}/test/license-test/run-license-test.sh

spellcheck:
	${MAKEFILE_PATH}/test/readme-test/run-readme-spellcheck

shellcheck:
	${MAKEFILE_PATH}/test/shellcheck/run-shellcheck

test: unit-test e2e-test license-test spellcheck shellcheck

fmt:
	goimports -w ./ && gofmt -s -w ./

homebrew-sync-dry-run:
	${MAKEFILE_PATH}/scripts/sync-to-aws-homebrew-tap -d -b ${BIN} -f ${REPO_SHORT_NAME} -r ${REPO_FULL_NAME} -p ${SUPPORTED_PLATFORMS} -v ${LATEST_RELEASE_TAG}

homebrew-sync:
	${MAKEFILE_PATH}/scripts/sync-to-aws-homebrew-tap -b ${BIN} -f ${REPO_SHORT_NAME} -r ${REPO_FULL_NAME} -p ${SUPPORTED_PLATFORMS}

## requires a github token
upload-resources-to-github:
	${MAKEFILE_PATH}/scripts/upload-resources-to-github

release: build-binaries build-docker-images upload-resources-to-github

help:
	@grep -E '^[a-zA-Z_-]+:.*$$' $(MAKEFILE_LIST) | sort

## Targets intended to be run in preparation for a new release
draft-release-notes:
	${MAKEFILE_PATH}/scripts/draft-release-notes

create-local-release-tag-major:
	${MAKEFILE_PATH}/scripts/create-local-tag-for-release -m

create-local-release-tag-minor:
	${MAKEFILE_PATH}/scripts/create-local-tag-for-release -i

create-local-release-tag-patch:
	${MAKEFILE_PATH}/scripts/create-local-tag-for-release -p

create-release-prep-pr:
	${MAKEFILE_PATH}/scripts/prepare-for-release

create-release-prep-pr-draft:
	${MAKEFILE_PATH}/scripts/prepare-for-release -d

release-prep-major: create-local-release-tag-major create-release-prep-pr

release-prep-minor: create-local-release-tag-minor create-release-prep-pr

release-prep-patch: create-local-release-tag-patch create-release-prep-pr

release-prep-custom: # Run make NEW_VERSION=v1.2.3 release-prep-custom to prep for a custom release version
ifdef NEW_VERSION
	$(shell echo "${MAKEFILE_PATH}/scripts/create-local-tag-for-release -v $(NEW_VERSION) && echo && make create-release-prep-pr")
endif