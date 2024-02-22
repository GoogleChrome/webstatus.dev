SHELL := /bin/bash
NPROCS := $(shell nproc)

.PHONY: all \
		clean \
		test \
		gen \
		openapi \
		jsonschema \
		lint \
		test \
		dev_workflows \
		precommit \
		minikube-delete \
		minikube-clean-restart \
		start-local \
		deploy-local \
		stop-local \
		port-forward-manual \
		port-forward-terminate

build: gen go-build node-install

clean: clean-gen clean-node port-forward-terminate minikube-delete

precommit: license-check lint test

################################
# Local Environment
################################
SKAFFOLD_FLAGS = -p local
SKAFFOLD_RUN_FLAGS = $(SKAFFOLD_FLAGS) --build-concurrency=$(NPROCS) --no-prune=false --cache-artifacts=false
start-local: configure-skaffold
	skaffold dev $(SKAFFOLD_RUN_FLAGS)

debug-local: configure-skaffold
	skaffold debug $(SKAFFOLD_RUN_FLAGS)

configure-skaffold: minikube-running
	skaffold config set --kube-context "$${MINIKUBE_PROFILE}" local-cluster true

deploy-local: configure-skaffold
	skaffold run $(SKAFFOLD_RUN_FLAGS) --status-check=true --port-forward=off

delete-local:
	skaffold delete $(SKAFFOLD_FLAGS) || true

port-forward-manual: port-forward-terminate
	kubectl wait --for=condition=ready pod/frontend
	kubectl wait --for=condition=ready pod/backend
	kubectl port-forward --address 127.0.0.1 pod/frontend 5555:5555 2>&1 >/dev/null &
	kubectl port-forward --address 127.0.0.1 pod/backend 8080:8080 2>&1 >/dev/null &

port-forward-terminate:
	pkill kubectl -9 || true

# Prerequisite target to start minikube if necessary
minikube-running:
		# Check if minikube is running using a shell command
		@if ! minikube status -p "$${MINIKUBE_PROFILE}" | grep -q "Running"; then \
				minikube start -p "$${MINIKUBE_PROFILE}"; \
		fi
minikube-clean-restart: minikube-delete minikube-running
minikube-delete:
	minikube delete -p "$${MINIKUBE_PROFILE}" || true

stop-local:
	minikube stop -p "$${MINIKUBE_PROFILE}"

################################
# Generated Files
################################
gen: openapi jsonschema

clean-gen: clean-openapi clean-jsonschema

################################
# Generated Files: From OpenAPI
################################
openapi: go-openapi node-openapi

clean-openapi: clean-go-openapi

OAPI_GEN_CONFIG = openapi/types.cfg.yaml
OPENAPI_OUT_DIR = lib/gen/openapi

# Pattern rule to generate types and server code for different packages
$(OPENAPI_OUT_DIR)/%/types.gen.go: openapi/%/openapi.yaml
	oapi-codegen -config $(OAPI_GEN_CONFIG) \
							 -o $(OPENAPI_OUT_DIR)/$*/types.gen.go -package $(shell basename $*) $<

$(OPENAPI_OUT_DIR)/%/server.gen.go: openapi/%/openapi.yaml
	oapi-codegen -config openapi/server.cfg.yaml \
							 -o $(OPENAPI_OUT_DIR)/$*/server.gen.go -package $(shell basename $*) $<

$(OPENAPI_OUT_DIR)/%/client.gen.go: openapi/%/openapi.yaml
	oapi-codegen -config openapi/client.cfg.yaml \
							 -o $(OPENAPI_OUT_DIR)/$*/client.gen.go -package $(shell basename $*) $<

# Target to generate all OpenAPI code
go-openapi: $(OPENAPI_OUT_DIR)/backend/types.gen.go \
			$(OPENAPI_OUT_DIR)/backend/server.gen.go \
			$(OPENAPI_OUT_DIR)/workflows/steps/web_feature_consumer/client.gen.go \
			$(OPENAPI_OUT_DIR)/workflows/steps/web_feature_consumer/types.gen.go \
			$(OPENAPI_OUT_DIR)/workflows/steps/web_feature_consumer/server.gen.go \
			$(OPENAPI_OUT_DIR)/workflows/steps/common/repo_downloader/client.gen.go \
			$(OPENAPI_OUT_DIR)/workflows/steps/common/repo_downloader/types.gen.go \
			$(OPENAPI_OUT_DIR)/workflows/steps/common/repo_downloader/server.gen.go

clean-go-openapi:
	rm -rf $(addprefix $(OPENAPI_OUT_DIR)/, */types.gen.go */server.gen.go)

node-openapi:
	npx openapi-typescript openapi/backend/openapi.yaml -o lib/gen/openapi/ts-webstatus.dev-backend-types/types.d.ts

# No need for a clean-node-openapi as it is covered by the `node-clean` target.

################################
# Generated Files: From JSONSchema
################################
JSONSCHEMA_OUT_DIR = lib/gen/jsonschema

download-schemas:
	wget -O jsonschema/web-platform-dx_web-features/defs.schema.json \
		https://raw.githubusercontent.com/web-platform-dx/feature-set/main/schemas/defs.schema.json

jsonschema:
	npx quicktype \
		--src jsonschema/web-platform-dx_web-features/defs.schema.json \
		--src-lang schema \
		--lang go \
		--top-level FeatureData \
		--out $(JSONSCHEMA_OUT_DIR)/web_platform_dx__web_features/feature_data.go \
		--package web_platform_dx__web_features \
		--field-tags json

clean-jsonschema:
	rm -rf $(JSONSCHEMA_OUT_DIR)/**/*.go

################################
# Lint
################################
golint-version:
	golangci-lint --version

lint: go-lint node-lint tf-lint shell-lint style-lint

go-lint: golint-version
	go list -f '{{.Dir}}/...' -m | xargs golangci-lint run

node-lint: node-install
	npm run lint -w frontend
	npx prettier . --check

tf-lint:
	terraform fmt -recursive -check .

shell-lint:
	shellcheck .devcontainer/*.sh
	shellcheck infra/**/*.sh

lint-fix: node-install
	npm run lint-fix -w frontend
	terraform fmt -recursive .
	npx prettier . --write
	npx stylelint "frontend/src/**/*.css" --fix

style-lint:
	npx stylelint "frontend/src/**/*.css"

################################
# Test
################################
test: go-test node-test

# Skip the module if it ends in 'lib/gen'
go-test:
	@declare -a GO_MODULES=(); \
	readarray -t GO_MODULES <  <(go list -f {{.Dir}} -m); \
	for GO_MODULE in $${GO_MODULES[@]}; \
	do \
		if [[ "$$GO_MODULE" != *"lib/gen" ]]; then \
			echo "********* Testing module: $${GO_MODULE} *********" ; \
			GO_COVERAGE_DIR="$${GO_MODULE}/coverage/unit" ; \
			mkdir -p $${GO_COVERAGE_DIR} ; \
			go test -cover -covermode=atomic -coverprofile=$${GO_COVERAGE_DIR}/cover.out "$${GO_MODULE}/..." && \
			echo "Generating coverage report for $${GO_MODULE}" && \
			go tool cover --func=$${GO_COVERAGE_DIR}/cover.out && \
			echo -e "\n\n" || exit 1; \
		fi \
	done

node-test: playwright-install
	npm run test -ws

################################
# License
################################
COPYRIGHT_NAME := Google LLC
# Description of ignored files
# lib/gen - all generated files
# .terraform.lock.hcl - generated lock file for terraform
# frontend/{dist|static|build} - built files, not source files that are checked in
# frontend/node_modules - External Node dependencies
# frontend/coverage - Generated html files for coverage
# playwright-report - Generated html files for playwright
# node_modules - External Node dependencies
ADDLICENSE_ARGS := -c "${COPYRIGHT_NAME}" \
	-l apache \
	-ignore 'lib/gen/**' \
	-ignore '**/.terraform.lock.hcl' \
	-ignore 'frontend/dist/**' \
	-ignore 'frontend/static/**' \
	-ignore 'frontend/node_modules/**' \
	-ignore 'frontend/coverage/**' \
	-ignore 'playwright-report/**' \
	-ignore 'node_modules/**'
download-addlicense:
	go install github.com/google/addlicense@latest

license-check: download-addlicense
	addlicense -check $(ADDLICENSE_ARGS) .

license-fix: download-addlicense
	addlicense $(ADDLICENSE_ARGS) .

################################
# Playwright
################################
fresh-env-for-playwright: playwright-install delete-local deploy-local port-forward-manual

playwright-update-snapshots: fresh-env-for-playwright
	npx playwright test --update-snapshots

playwright-install:
	npx playwright install --with-deps

playwright-test: fresh-env-for-playwright
	npx playwright test

################################
# Go Misc
################################

go-tidy:
	go list -f '{{.Dir}}/...' -m | xargs go mod tidy
go-build: # TODO: Add go-tidy here once we move to GitHub.
	go list -f '{{.Dir}}/...' -m | xargs go build

################################
# Node Misc
################################
node-install:
	npm install -ws --foreground-scripts

clean-node:
	npm run clean -ws

################################
# Local Data / Workflows
################################
dev_workflows: web_feature_local_workflow
web_feature_local_workflow: FLAGS := -repo_downloader_host=http://localhost:8091 -web_consumer_host=http://localhost:8092
web_feature_local_workflow:
	go run ./util/cmd/local_web_feature_workflow/main.go $(FLAGS)
