MAKEFILE_PATH = $(dir $(realpath -s $(firstword $(MAKEFILE_LIST))))
PROJECT_IMPORT_DIR = ez-ec2
BUILD_DIR_PATH = ${MAKEFILE_PATH}/build
CLI_BINARY_NAME = ez-ec2
VERSION ?= $(shell git describe --tags --always --dirty)
BIN ?= ez-ec2
REPO_FULL_NAME ?= awslabs/aws-simple-ec2-cli
GOOS ?= $(uname | tr '[:upper:]' '[:lower:]')
GOARCH ?= amd64
GOPROXY ?= "https://proxy.golang.org,direct"
SUPPORTED_PLATFORMS ?= "windows/amd64,darwin/amd64,linux/amd64,linux/arm64,linux/arm"
SELECTOR_PKG_VERSION_VAR=github.com/awslabs/aws-simple-ec2-cli/v2/pkg/selector.versionID
LATEST_RELEASE_TAG=$(shell git tag | tail -1)
PREVIOUS_RELEASE_TAG=$(shell git tag | tail -2 | head -1)

# The main CloudFormation template for creating a new stack during launch
EZEC2_CLOUDFORMATION_TEMPLATE_FILE=${MAKEFILE_PATH}/cloudformation_template.json
EZEC2_CLOUDFORMATION_TEMPLATE_ENCODED=$(shell cat ${EZEC2_CLOUDFORMATION_TEMPLATE_FILE} | base64 | tr -d \\n)
EZEC2_CLOUDFORMATION_TEMPLATE_VAR=${PROJECT_IMPORT_DIR}/pkg/cfn.Ezec2CloudformationTemplateEncoded

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

EMBED_TEMPLATE_FLAG=-ldflags '-X "${EZEC2_CLOUDFORMATION_TEMPLATE_VAR}=${EZEC2_CLOUDFORMATION_TEMPLATE_ENCODED}"\
 -X "${E2E_CFN_TEST_CLOUDFORMATION_TEMPLATE_VAR}=${E2E_CFN_TEST_CLOUDFORMATION_TEMPLATE_ENCODED}"\
 -X "${E2E_CONNECT_TEST_CLOUDFORMATION_TEMPLATE_VAR}=${E2E_CONNECT_TEST_CLOUDFORMATION_TEMPLATE_ENCODED}"\
 -X "${E2E_EC2HELPER_TEST_CLOUDFORMATION_TEMPLATE_VAR}=${E2E_EC2HELPER_TEST_CLOUDFORMATION_TEMPLATE_ENCODED}"'

E2E_TEST_PACKAGES=ez-ec2/test/e2e/...

GO_TEST=go test ${EMBED_TEMPLATE_FLAG} -bench=. ${MAKEFILE_PATH}
DELETE_STACK=aws cloudformation delete-stack --stack-name

$(shell mkdir -p ${BUILD_DIR_PATH} && touch ${BUILD_DIR_PATH}/_go.mod)

clean:
	rm -rf ${BUILD_DIR_PATH}/ && go clean -testcache ./...

compile:
	go build ${EMBED_TEMPLATE_FLAG} -o ${BUILD_DIR_PATH}/${CLI_BINARY_NAME} ${MAKEFILE_PATH}/main.go

build: clean compile

build-binaries:
	${MAKEFILE_PATH}/scripts/build-binaries -p ${SUPPORTED_PLATFORMS} -v ${VERSION}

unit-test:
	${GO_TEST}/pkg/... -v -coverprofile=coverage.out -covermode=atomic -outputdir=${BUILD_DIR_PATH}; go tool cover -func ${BUILD_DIR_PATH}/coverage.out

e2e-test:
	${GO_TEST}/test/e2e/... -v
	${DELETE_STACK}ez-ec2-e2e-cfn-test
	${DELETE_STACK}ez-ec2-e2e-connect-test
	${DELETE_STACK}ez-ec2-e2e-ec2helper-test

e2e-cfn-test:
	${GO_TEST}/test/e2e/e2e-cfn-test/... -v
	${DELETE_STACK}ez-ec2-e2e-cfn-test

e2e-connect-test:
	${GO_TEST}/test/e2e/e2e-connect-test/... -v
	${DELETE_STACK}ez-ec2-e2e-connect-test

e2e-ec2helper-test:
	${GO_TEST}/test/e2e/e2e-ec2helper-test/... -v
	${DELETE_STACK}ez-ec2-e2e-ec2helper-test

test: unit-test e2e-test