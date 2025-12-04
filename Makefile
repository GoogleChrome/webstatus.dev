SHELL := /bin/bash
NPROCS := $(shell nproc)
GH_REPO := "GoogleChrome/webstatus.dev"

DOCKERFILES := \
	images/go_service.Dockerfile \
	images/nodejs_service.Dockerfile \
	otel/Dockerfile \
	./.dev/auth/Dockerfile \
	./.dev/datastore/Dockerfile \
	./.dev/spanner/Dockerfile \
	./.dev/valkey/Dockerfile \
	./.devcontainer/Dockerfile

.PHONY: all \
		antlr-gen \
		clean \
		test \
		gen \
		clean-openapi \
		go-openapi \
		node-openapi \
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
		port-forward-terminate \
		unstaged-changes

build: gen go-build node-install

clean: clean-gen clean-node port-forward-terminate minikube-delete

precommit: license-check lint unstaged-changes test

################################
# Local Environment
################################
SKAFFOLD_FLAGS = -p local
SKAFFOLD_RUN_FLAGS = $(SKAFFOLD_FLAGS) --build-concurrency=$(NPROCS) --no-prune=false --cache-artifacts=false --port-forward=off
start-local: configure-skaffold gen
	skaffold dev $(SKAFFOLD_RUN_FLAGS)

debug-local: configure-skaffold gen
	skaffold debug $(SKAFFOLD_RUN_FLAGS)

configure-skaffold: minikube-running
	skaffold config set --kube-context "$${MINIKUBE_PROFILE}" local-cluster true

deploy-local: configure-skaffold
	skaffold run $(SKAFFOLD_RUN_FLAGS) --status-check=true

delete-local:
	skaffold delete $(SKAFFOLD_FLAGS) || true

define wait_for_port
    @echo "Waiting for $(2) on port $(1) to respond..."
    @for i in $$(seq 1 5); do \
        if nc -zvw 2 localhost $(1); then \
            echo "$(2) on port $(1) is responding."; \
            exit 0; \
        fi; \
        if [ $$i -lt 5 ]; then \
            echo "Port $(1) ($(2)) not responding (attempt $$i/5). Retrying in 3 seconds..."; \
            sleep 3; \
        else \
            echo "Error: $(2) on port $(1) did not respond after 5 attempts."; \
            exit 1; \
        fi; \
    done
endef


check-local-ports:
	$(call wait_for_port,5555,frontend)
	$(call wait_for_port,8080,backend)
	$(call wait_for_port,9099,auth-main)
	$(call wait_for_port,9100,auth-aux)
	$(call wait_for_port,9010,spanner)
	$(call wait_for_port,8086,datastore)
	$(call wait_for_port,8087,wiremock)


port-forward-manual: port-forward-terminate
	kubectl wait --for=condition=ready pod/frontend
	kubectl wait --for=condition=ready pod/backend
	kubectl wait --for=condition=ready pod/auth
	kubectl wait --for=condition=ready pod/datastore
	kubectl wait --for=condition=ready pod/spanner
	kubectl wait --for=condition=ready pod/wiremock
	kubectl port-forward --address 127.0.0.1 pod/frontend 5555:5555 2>&1 >/dev/null &
	kubectl port-forward --address 127.0.0.1 pod/backend 8080:8080 2>&1 >/dev/null &
	kubectl port-forward --address 127.0.0.1 pod/auth 9099:9099 2>&1 >/dev/null &
	kubectl port-forward --address 127.0.0.1 pod/auth 9100:9100 2>&1 >/dev/null &
	kubectl port-forward --address 127.0.0.1 pod/spanner 9010:9010 2>&1 >/dev/null &
	kubectl port-forward --address 127.0.0.1 pod/datastore 8086:8086 2>&1 >/dev/null &
	kubectl port-forward --address 127.0.0.1 pod/wiremock 8087:8080 2>&1 >/dev/null &
	make check-local-ports

port-forward-terminate:
	fuser -k 5555/tcp || true
	fuser -k 8080/tcp || true
	fuser -k 9099/tcp || true
	fuser -k 9100/tcp || true
	fuser -k 9010/tcp || true
	fuser -k 8086/tcp || true
	fuser -k 8087/tcp || true

# Prerequisite target to start minikube if necessary
minikube-running:
		# Check if minikube is running using a shell command
		@if ! minikube status -p "$${MINIKUBE_PROFILE}" | grep -q "Running"; then \
				minikube start -p "$${MINIKUBE_PROFILE}" --cni calico --disk-size=10gb --cpus=2 --memory=6g; \
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
openapi: clean-openapi go-openapi node-openapi

clean-openapi: clean-go-openapi

OAPI_GEN_CONFIG = openapi/types.cfg.yaml
OPENAPI_OUT_DIR = lib/gen/openapi

# Pattern rule to generate types and server code for different packages
$(OPENAPI_OUT_DIR)/%/types.gen.go: openapi/%/openapi.yaml go-install-tools
	go tool oapi-codegen -config $(OAPI_GEN_CONFIG) \
							 -o $(OPENAPI_OUT_DIR)/$*/types.gen.go -package $(shell basename $*) $<

$(OPENAPI_OUT_DIR)/%/server.gen.go: openapi/%/openapi.yaml go-install-tools
	go tool oapi-codegen -config openapi/server.cfg.yaml \
							 -o $(OPENAPI_OUT_DIR)/$*/server.gen.go -package $(shell basename $*) $<

$(OPENAPI_OUT_DIR)/%/client.gen.go: openapi/%/openapi.yaml go-install-tools
	go tool oapi-codegen -config openapi/client.cfg.yaml \
							 -o $(OPENAPI_OUT_DIR)/$*/client.gen.go -package $(shell basename $*) $<

# Target to generate all OpenAPI code
go-openapi: $(OPENAPI_OUT_DIR)/backend/types.gen.go \
			$(OPENAPI_OUT_DIR)/backend/server.gen.go

clean-go-openapi:
	rm -rf $(addprefix $(OPENAPI_OUT_DIR)/backend, /types.gen.go /server.gen.go)

node-openapi:
	npx openapi-typescript openapi/backend/openapi.yaml -o lib/gen/openapi/ts-webstatus.dev-backend-types/types.d.ts

# No need for a clean-node-openapi as it is covered by the `node-clean` target.

################################
# Generated Files: From JSONSchema
################################
JSONSCHEMA_OUT_DIR = lib/gen/jsonschema

download-schemas:
	wget -O jsonschema/web-platform-dx_web-features/v3.data.schema.json \
		https://raw.githubusercontent.com/web-platform-dx/web-features/refs/heads/main/schemas/data.schema.json
	wget -O jsonschema/mdn_browser-compat-data/browsers.schema.json \
		https://raw.githubusercontent.com/mdn/browser-compat-data/main/schemas/browsers.schema.json
	wget -O jsonschema/web-platform-dx_web-features-mappings/combined-schema.gen.json \
		https://raw.githubusercontent.com/web-platform-dx/web-features-mappings/refs/heads/main/combined-schema.gen.json

jsonschema: clean-jsonschema
	npx quicktype \
		--src jsonschema/web-platform-dx_web-features/v3.data.schema.json \
		--src-lang schema \
		--lang go \
		--top-level FeatureData \
		--out $(JSONSCHEMA_OUT_DIR)/web_platform_dx__web_features_v3/feature_data.go \
		--package web_platform_dx__web_features_v3 \
		--field-tags json

	npx quicktype \
		--src jsonschema/mdn_browser-compat-data/browsers.schema.json \
		--src-lang schema \
		--lang go \
		--top-level BrowserData \
		--out $(JSONSCHEMA_OUT_DIR)/mdn__browser_compat_data/browser_data.go \
		--package mdn__browser_compat_data \
		--field-tags json

	npx quicktype \
		--src jsonschema/web-platform-dx_web-features-mappings/combined-schema.gen.json \
		--src-lang schema \
		--lang go \
		--top-level WebFeaturesMappings \
		--out $(JSONSCHEMA_OUT_DIR)/web_platform_dx__web_features_mappings/web_features_mappings.go \
		--package web_platform_dx__web_features_mappings \
		--field-tags json

	npx quicktype \
		--src jsonschema/web-platform-dx_web-features-mappings/combined-schema.gen.json \
		--src-lang schema \
		--lang ts \
		--top-level webFeaturesMapping \
		--out lib/gen/jsonschema/web-platform-dx_web-features-mappings-ts/types.d.ts \
		--prefer-unions \
		--just-types


clean-jsonschema:
	rm -rf $(JSONSCHEMA_OUT_DIR)/**/*.go
	rm -rf $(JSONSCHEMA_OUT_DIR)/**/*.ts

################################
# Lint
################################
golint-version:
	golangci-lint --version

lint: go-lint node-lint tf-lint shell-lint style-lint dockerfile-lint

go-lint: golint-version go-workspace-setup
	go list -f '{{.Dir}}/...' -m | xargs -t golangci-lint run

node-lint: node-install
	npm run lint -w frontend
	npx prettier . --check

# Need to clean out the .terraform directory before linting.
tf-lint:
	rm -rf infra/.terraform
	cd infra && terraform init -backend=false -reconfigure --var-file=.envs/staging.tfvars --backend-config=.envs/backend-staging.tfvars && terraform validate
	cd infra && terraform init -backend=false -reconfigure --var-file=.envs/prod.tfvars --backend-config=.envs/backend-prod.tfvars && terraform validate
	terraform fmt -recursive -check .

shell-lint:
	shellcheck .devcontainer/*.sh
	shellcheck infra/**/*.sh
	shellcheck .dev/**/*.sh
	shellcheck util/*.sh
	shellcheck util/deployment/*

lint-fix: node-install
	npm run lint-fix -w frontend
	terraform fmt -recursive .
	npx prettier . --write
	npx stylelint "frontend/src/**/*.css" --fix

style-lint:
	npx stylelint "frontend/src/**/*.css"

dockerfile-lint:
	@$(foreach dockerfile,$(DOCKERFILES),docker build --check -f $(dockerfile) .;)

################################
# Test
################################
test: go-test node-test

# Clean up any dangling test containers
clean-up-go-testcontainers:
	docker rm -f webstatus-dev-test-valkey webstatus-dev-test-datastore webstatus-dev-test-spanner
# TODO. We run the tests sequentially with `-p 1` because the testcontainers
# do not play nicely together when running in parallel and take a long time to
# reconcile state. Once the testcontainers library becomes stable (goes v1.0.0),
# we should remove the `-p 1`.
go-test: clean-up-go-testcontainers go-workspace-setup
	@declare -a GO_MODULES=(); \
	readarray -t GO_MODULES <  <(go list -f {{.Dir}} -m); \
	for GO_MODULE in $${GO_MODULES[@]}; \
	do \
		if [[ "$$GO_MODULE" != *"lib/gen" && "$$GO_MODULE" != *"tools"*  ]]; then \
			echo "********* Testing module: $${GO_MODULE} *********" ; \
			GO_COVERAGE_DIR="$${GO_MODULE}/coverage/unit" ; \
			mkdir -p $${GO_COVERAGE_DIR} ; \
			go test -race -p 1 -cover -covermode=atomic -coverprofile=$${GO_COVERAGE_DIR}/cover.out "$${GO_MODULE}/..." && \
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
# frontend/placeholder/static/index.html - Temporary placeholder
# playwright-report - Generated html files for playwright
# node_modules - External Node dependencies
# infra/storage/spanner/schema.sql - Empty base schema. Wrench does not like an empty schema with comments.
# antlr/.antlr - for intermediate antlr files
# .devcontainer/cache - cached files
ADDLICENSE_ARGS := -c "${COPYRIGHT_NAME}" \
	-l apache \
	-ignore 'lib/gen/**' \
	-ignore '**/.terraform.lock.hcl' \
	-ignore 'frontend/build/**' \
	-ignore 'frontend/dist/**' \
	-ignore 'frontend/placeholder/static/index.html' \
	-ignore 'frontend/static/**' \
	-ignore 'frontend/node_modules/**' \
	-ignore 'frontend/coverage/**' \
	-ignore 'playwright-report/**' \
	-ignore 'node_modules/**' \
	-ignore 'infra/storage/spanner/schema.sql' \
	-ignore 'antlr/.antlr/**' \
	-ignore '.devcontainer/cache/**'

license-check: go-install-tools
	go tool addlicense -check $(ADDLICENSE_ARGS) .

license-fix: go-install-tools
	go tool addlicense $(ADDLICENSE_ARGS) .

unstaged-changes:
	git diff --exit-code

################################
# Playwright
################################
# Set this variable to any non-empty value (e.g., SKIP_FRESH_ENV=1) to skip the
# fresh-env-for-playwright prerequisite. If unset, the fresh environment will be created.
SKIP_FRESH_ENV ?=

fresh-env-for-playwright: $(if $(SKIP_FRESH_ENV),,playwright-install delete-local build deploy-local port-forward-manual dev_fake_users dev_fake_data)

playwright-install:
	npx playwright install --with-deps

playwright-update-snapshots: fresh-env-for-playwright
	npx playwright test --update-snapshots

playwright-test: fresh-env-for-playwright
	npx playwright test

playwright-ui: fresh-env-for-playwright
	npx playwright test --ui --ui-port=8123

playwright-debug: fresh-env-for-playwright
	npx playwright test --debug --ui-port=8123

playwright-open-report:
	npx playwright show-report playwright-report --host 0.0.0.0

playwright-show-traces:
	find playwright-report/data -name *.zip -printf "%TY-%Tm-%Td %TH:%TM:%TS %Tz %p\n"

################################
# Go Misc
################################
go-install-tools: go-workspace-setup
	pushd tools && \
	go mod download && \
	popd

# Currently, the tools will not upgrade automatically with the `go-update
# target. Instead, we need to have a separate target
go-update-tools: go-workspace-setup
	pushd tools && \
	go get -u github.com/cloudspannerecosystem/wrench@latest && \
	go get -u github.com/google/addlicense@latest && \
	go get github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen && \
	popd

go-update: go-workspace-setup
	@declare -a GO_MODULES=(); \
	readarray -t GO_MODULES <  <(go list -f {{.Dir}} -m); \
	for GO_MODULE in $${GO_MODULES[@]}; \
	do \
		echo "********* go get -u ./... module: $${GO_MODULE} *********" ; \
		pushd $${GO_MODULE} && \
		go get -u ./... && \
		echo -e "\n" || exit 1; \
		popd ; \
	done

go-tidy: go-workspace-setup
	@declare -a GO_MODULES=(); \
	readarray -t GO_MODULES <  <(go list -f {{.Dir}} -m); \
	for GO_MODULE in $${GO_MODULES[@]}; \
	do \
		echo "********* go mod tidy module: $${GO_MODULE} *********" ; \
		pushd $${GO_MODULE} && \
		go mod tidy && \
		echo -e "\n" || exit 1; \
		popd ; \
	done
go-build: go-workspace-setup go-tidy
	go list -f '{{.Dir}}/...' -m | xargs go build
go-workspace-setup: go-workspace-clean
	go work init && \
		go work use ./backend && \
		go work use ./lib && \
		go work use ./lib/gen && \
		go work use ./tools && \
		go work use ./util && \
		go work use ./workers/chime && \
		go work use ./workflows/steps/services/bcd_consumer && \
		go work use ./workflows/steps/services/chromium_histogram_enums && \
		go work use ./workflows/steps/services/developer_signals_consumer && \
		go work use ./workflows/steps/services/uma_export && \
		go work use ./workflows/steps/services/web_feature_consumer && \
		go work use ./workflows/steps/services/web_features_mapping_consumer && \
		go work use ./workflows/steps/services/wpt_consumer
go-workspace-clean:
	rm -rf go.work && rm -rf go.work.sum

################################
# Node Misc
################################
# Install all the packages from all workspaces.
node-install:
	npm ci --include-workspace-root=true --foreground-scripts=true

node-update:
	npm update
	npm update -w frontend

clean-node:
	npm run clean -ws

################################
# Local Data / Workflows
################################
dev_workflows: bcd_workflow web_feature_workflow web_features_mapping_workflow developer_signals_workflow chromium_histogram_enums_workflow wpt_workflow
web_feature_workflow:
	./util/run_job.sh web-features-consumer images/go_service.Dockerfile workflows/steps/services/web_feature_consumer \
		workflows/steps/services/web_feature_consumer/manifests/job.yaml web-features-consumer
developer_signals_workflow:
	./util/run_job.sh developer-signals-consumer images/go_service.Dockerfile workflows/steps/services/developer_signals_consumer \
		workflows/steps/services/developer_signals_consumer/manifests/job.yaml developer-signals-consumer
wpt_workflow:
	./util/run_job.sh wpt-consumer images/go_service.Dockerfile workflows/steps/services/wpt_consumer \
		workflows/steps/services/wpt_consumer/manifests/job.yaml wpt-consumer
bcd_workflow:
	./util/run_job.sh bcd-consumer images/go_service.Dockerfile workflows/steps/services/bcd_consumer \
		workflows/steps/services/bcd_consumer/manifests/job.yaml bcd-consumer
chromium_histogram_enums_workflow:
	./util/run_job.sh chromium-histogram-enums-consumer images/go_service.Dockerfile workflows/steps/services/chromium_histogram_enums \
		workflows/steps/services/chromium_histogram_enums/manifests/job.yaml chromium-histogram-enums-consumer
web_features_mapping_workflow:
	./util/run_job.sh web-features-mapping-consumer images/go_service.Dockerfile workflows/steps/services/web_features_mapping_consumer \
		workflows/steps/services/web_features_mapping_consumer/manifests/job.yaml web-features-mapping-consumer
dev_fake_users: build
	fuser -k 9099/tcp || true
	kubectl port-forward --address 127.0.0.1 pod/auth 9099:9099 2>&1 >/dev/null &
	go run util/cmd/load_test_users/main.go -project=local
dev_fake_data: build is_local_migration_ready check-local-ports
	SPANNER_EMULATOR_HOST=localhost:9010 DATASTORE_EMULATOR_HOST=localhost:8086 FIREBASE_AUTH_EMULATOR_HOST=localhost:9099 \
		go run ./util/cmd/load_fake_data/main.go \
			-spanner_project=local \
			-spanner_instance=local \
			-spanner_database=local \
			-datastore_project=local \
			$(LOAD_FAKE_DATA_FLAGS)
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
spanner_new_migration: go-install-tools
	go tool wrench migrate create --directory infra/storage/spanner

spanner_port_forward: spanner_port_forward_terminate
	kubectl wait --for=condition=ready pod/spanner
	kubectl port-forward --address 127.0.0.1 pod/spanner 9010:9010 2>&1 >/dev/null &

spanner_port_forward_terminate:
	fuser -k 9010/tcp || true

# For now install tbls when we absolutely need it.
# It is a heavy install.
spanner_er_diagram: spanner_port_forward
	go install github.com/k1LoW/tbls@v1.76.0
	SPANNER_EMULATOR_HOST=localhost:9010 tbls doc --rm-dist
	make spanner_port_forward_terminate

################################
# GitHub
################################
check-gh-login:
	@if ! gh auth status 2>/dev/null; then \
		echo "Not logged into GitHub CLI. Please run 'gh auth login'." && exit 1; \
	fi

print-gh-runs: check-gh-login
	gh run ls -R $(GH_REPO) -u $$(gh api user | jq -r '.login')

download-playwright-report-from-run-%: check-gh-login
	rm -rf playwright-report
	mkdir -p playwright-report
	gh run download -R $(GH_REPO) $* -n playwright-report --dir playwright-report
