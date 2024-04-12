SHELL := /bin/bash
NPROCS := $(shell nproc)

.PHONY: all \
		antlr-gen \
		clean \
		test \
		gen \
		openapi \
		jsonschema \
		lint \
		test \
		is_local_migration_ready \
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
SKAFFOLD_RUN_FLAGS = $(SKAFFOLD_FLAGS) --build-concurrency=$(NPROCS) --no-prune=false --cache-artifacts=false --port-forward=off
start-local: configure-skaffold
	skaffold dev $(SKAFFOLD_RUN_FLAGS)

debug-local: configure-skaffold
	skaffold debug $(SKAFFOLD_RUN_FLAGS)

configure-skaffold: minikube-running
	skaffold config set --kube-context "$${MINIKUBE_PROFILE}" local-cluster true

deploy-local: configure-skaffold
	skaffold run $(SKAFFOLD_RUN_FLAGS) --status-check=true

delete-local:
	skaffold delete $(SKAFFOLD_FLAGS) || true

port-forward-manual: port-forward-terminate
	kubectl wait --for=condition=ready pod/frontend
	kubectl wait --for=condition=ready pod/backend
	kubectl wait --for=condition=ready pod/web-feature-consumer
	kubectl port-forward --address 127.0.0.1 pod/frontend 5555:5555 2>&1 >/dev/null &
	kubectl port-forward --address 127.0.0.1 pod/backend 8080:8080 2>&1 >/dev/null &
	kubectl port-forward --address 127.0.0.1 pod/web-feature-consumer 8092:8080 2>&1 >/dev/null &
	curl -s -o /dev/null -m 5 http://localhost:8080 || true
	curl -s -o /dev/null -m 5 http://localhost:5555 || true
	curl -s -o /dev/null -m 5 http://localhost:8092 || true

port-forward-terminate:
	fuser -k 5555/tcp || true
	fuser -k 8080/tcp || true
	fuser -k 8092/tcp || true

# Prerequisite target to start minikube if necessary
minikube-running:
		# Check if minikube is running using a shell command
		@if ! minikube status -p "$${MINIKUBE_PROFILE}" | grep -q "Running"; then \
				minikube start -p "$${MINIKUBE_PROFILE}" --disk-size=10gb --cpus=2 --memory=4096m; \
		fi
minikube-clean-restart: minikube-delete minikube-running
minikube-delete:
	minikube delete -p "$${MINIKUBE_PROFILE}" || true

stop-local:
	minikube stop -p "$${MINIKUBE_PROFILE}"

################################
# Generated Files
################################
gen: openapi jsonschema antlr-gen

clean-gen: clean-openapi clean-jsonschema clean-antlr

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
	wget -O jsonschema/mdn_browser-compat-data/browsers.schema.json \
		https://raw.githubusercontent.com/mdn/browser-compat-data/main/schemas/browsers.schema.json

jsonschema:
	npx quicktype \
		--src jsonschema/web-platform-dx_web-features/defs.schema.json \
		--src-lang schema \
		--lang go \
		--top-level FeatureData \
		--out $(JSONSCHEMA_OUT_DIR)/web_platform_dx__web_features/feature_data.go \
		--package web_platform_dx__web_features \
		--field-tags json

	npx quicktype \
		--src jsonschema/mdn_browser-compat-data/browsers.schema.json \
		--src-lang schema \
		--lang go \
		--top-level BrowserData \
		--out $(JSONSCHEMA_OUT_DIR)/mdn__browser_compat_data/browser_data.go \
		--package mdn__browser_compat_data \
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
	shellcheck .dev/**/*.sh
	shellcheck util/*.sh

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
# ANTLR
################################
antlr-gen: clean-antlr
	java -jar /usr/local/lib/antlr-$${ANTLR4_VERSION}-complete.jar -Dlanguage=Go -o lib/gen/featuresearch/parser -visitor -no-listener antlr/FeatureSearch.g4

clean-antlr:
	rm -rf lib/gen/featuresearch/parser

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
# infra/storage/spanner/schema.sql - Empty base schema. Wrench does not like an empty schema with comments.
# antlr/.antlr - for intermediate antlr files
ADDLICENSE_ARGS := -c "${COPYRIGHT_NAME}" \
	-l apache \
	-ignore 'lib/gen/**' \
	-ignore '**/.terraform.lock.hcl' \
	-ignore 'frontend/dist/**' \
	-ignore 'frontend/static/**' \
	-ignore 'frontend/node_modules/**' \
	-ignore 'frontend/coverage/**' \
	-ignore 'playwright-report/**' \
	-ignore 'node_modules/**' \
	-ignore 'infra/storage/spanner/schema.sql' \
	-ignore 'antlr/.antlr/**'
download-addlicense:
	go install github.com/google/addlicense@latest

license-check: download-addlicense
	addlicense -check $(ADDLICENSE_ARGS) .

license-fix: download-addlicense
	addlicense $(ADDLICENSE_ARGS) .

################################
# Playwright
################################
fresh-env-for-playwright: playwright-install delete-local deploy-local dev_fake_data port-forward-manual

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
dev_workflows: bcd_workflow web_feature_local_workflow wpt_workflow
web_feature_local_workflow: FLAGS := -web_consumer_host=http://localhost:8092
web_feature_local_workflow: port-forward-manual
	go run ./util/cmd/local_web_feature_workflow/main.go $(FLAGS)
wpt_workflow:
	./util/run_job.sh wpt-consumer images/go_service.Dockerfile workflows/steps/services/wpt_consumer \
		workflows/steps/services/wpt_consumer/manifests/job.yaml wpt-consumer
bcd_workflow:
	./util/run_job.sh bcd-consumer images/go_service.Dockerfile workflows/steps/services/bcd_consumer \
		workflows/steps/services/bcd_consumer/manifests/job.yaml bcd-consumer
dev_fake_data: is_local_migration_ready
	fuser -k 9010/tcp || true
	kubectl port-forward --address 127.0.0.1 pod/spanner 9010:9010 2>&1 >/dev/null &
	SPANNER_EMULATOR_HOST=localhost:9010 go run ./util/cmd/load_fake_data/main.go -spanner_project=local -spanner_instance=local -spanner_database=local
	fuser -k 9010/tcp || true
is_local_migration_ready:
	kubectl wait --for=condition=ready --timeout=300s pod/spanner
	@MAX_RETRIES=5; SLEEP_INTERVAL=5 ; \
    for (( i=0; i < $$MAX_RETRIES; i++ )); do \
		[[ $$(kubectl exec pods/spanner -- wrench migrate version) -eq 1 ]] && break; \
		echo "Migration not ready (attempt $$i). Retrying in $$SLEEP_INTERVAL seconds..."; sleep $$SLEEP_INTERVAL ; \
    done


################################
# Spanner Management
################################
spanner_new_migration:
	wrench migrate create --directory infra/storage/spanner

spanner_port_forward: spanner_port_forward_terminate
	kubectl wait --for=condition=ready pod/spanner
	kubectl port-forward --address 127.0.0.1 pod/spanner 9010:9010 2>&1 >/dev/null &

spanner_port_forward_terminate:
	fuser -k 9010/tcp || true

# For now install tbls when we absolutely need it.
# It is a heavy install.
spanner_er_diagram: spanner_port_forward
	go install github.com/k1LoW/tbls@v1.73.2
	SPANNER_EMULATOR_HOST=localhost:9010 tbls doc --rm-dist
	make spanner_port_forward_terminate
