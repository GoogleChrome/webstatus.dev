SHELL := /bin/bash

.PHONY: all clean test gen openapi lint

gen: openapi jsonschema

################################
# OpenAPI Generation
################################
openapi: go-openapi node-openapi

OAPI_GEN_CONFIG = openapi/types.cfg.yaml
OUT_DIR = lib/gen/openapi

# Pattern rule to generate types and server code for different packages
$(OUT_DIR)/%/types.gen.go: openapi/%/openapi.yaml
	oapi-codegen -config $(OAPI_GEN_CONFIG) \
	             -o $(OUT_DIR)/$*/types.gen.go -package $(shell basename $*) $<

$(OUT_DIR)/%/server.gen.go: openapi/%/openapi.yaml
	oapi-codegen -config openapi/server.cfg.yaml \
	             -o $(OUT_DIR)/$*/server.gen.go -package $(shell basename $*) $<

# Target to generate all OpenAPI code
go-openapi: $(OUT_DIR)/backend/types.gen.go \
            $(OUT_DIR)/backend/server.gen.go \
            $(OUT_DIR)/workflows/steps/web_feature_consumer/types.gen.go \
            $(OUT_DIR)/workflows/steps/web_feature_consumer/server.gen.go \
            $(OUT_DIR)/workflows/steps/common/repo_downloader/types.gen.go \
            $(OUT_DIR)/workflows/steps/common/repo_downloader/server.gen.go

node-openapi:
	npx openapi-typescript openapi/backend/openapi.yaml -o lib/gen/openapi/ts-webstatus.dev-backend-types/types.d.ts

################################
# JSON Schema
################################

download-schemas:
	wget -O jsonschema/web-platform-dx_web-features/defs.schema.json \
		https://raw.githubusercontent.com/web-platform-dx/feature-set/main/schemas/defs.schema.json

jsonschema:
	quicktype \
		--src jsonschema/web-platform-dx_web-features/defs.schema.json \
		--src-lang schema \
		--lang go \
		--top-level FeatureData \
		--out lib/gen/jsonschema/web_platform_dx__web_features/feature_data.go \
		--package web_platform_dx__web_features \
		--field-tags json

################################
# Lint
################################
golint-version:
	golangci-lint --version

lint: go-lint node-lint tf-lint shell-lint

go-lint: golint-version
	go list -f '{{.Dir}}/...' -m | xargs golangci-lint run

node-lint: frontend-deps
	npm run lint -w frontend
	npx prettier . --check

tf-lint:
	terraform fmt -recursive -check .

shell-lint:
	shellcheck .devcontainer/*.sh
	shellcheck infra/**/*.sh

################################
# Test
################################
unit-test:
	@declare -a GO_MODULES=(); \
	readarray -t GO_MODULES <  <(go list -f {{.Dir}} -m); \
	for GO_MODULE in $${GO_MODULES[@]}; \
	do \
		echo "********* Testing module: $${GO_MODULE} *********" ; \
		GO_COVERAGE_DIR="$${GO_MODULE}/coverage/unit" ; \
		mkdir -p $${GO_COVERAGE_DIR} ; \
		go test -cover -covermode=atomic -coverprofile=$${GO_COVERAGE_DIR}/cover.out "$${GO_MODULE}/..."; \
		echo "Generating coverage report for $${GO_MODULE}" ; \
		go tool cover --func=$${GO_COVERAGE_DIR}/cover.out ; \
		echo -e "\n\n" ; \
	done

lint-fix: frontend-deps
	npm run lint-fix -w frontend
	terraform fmt -recursive .
	npx prettier . --write

################################
# License
################################
COPYRIGHT_NAME := Google LLC
# Description of ignored files
# lib/gen - all generated files
# .terraform.lock.hcl - generated lock file for terraform
# frontend/{dist|static|build} - built files, not source files that are checked in
# frontend/node_modules - External Node dependencies
# node_modules - External Node dependencies
ADDLICENSE_ARGS := -c "${COPYRIGHT_NAME}" \
	-l apache \
	-ignore 'lib/gen/**' \
	-ignore '**/.terraform.lock.hcl' \
	-ignore 'frontend/dist/**' \
	-ignore 'frontend/static/**' \
	-ignore 'frontend/node_modules/**' \
	-ignore 'node_modules/**'
download-addlicense:
	go install github.com/google/addlicense@latest

license-check: download-addlicense
	addlicense -check $(ADDLICENSE_ARGS) .

license-fix: download-addlicense
	addlicense $(ADDLICENSE_ARGS) .

################################
# Misc
################################
frontend-deps:
	npm install -w frontend
