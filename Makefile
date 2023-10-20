COPYRIGHT_NAME := Google LLC
# Description of ignored files
# lib/gen - all generated files
# .terraform.lock.hcl - generated lock file for terraform
# frontend/dist - built files, not source files that are checked in
# frontend/static - built files, not source files that are checked in
# frontend/node_modules - External Node dependencies
# node_modules - External Node dependencies
ADDLICENSE_ARGS := -c "${COPYRIGHT_NAME}" \
	-l apache \
	-ignore 'lib/gen' \
	-ignore '**/.terraform.lock.hcl' \
	-ignore 'frontend/dist/**' \
	-ignore 'frontend/static/**' \
	-ignore 'frontend/node_modules/**' \
	-ignore 'node_modules/**'
gen: openapi jsonschema

openapi:
	oapi-codegen -config openapi/types.cfg.yaml -o lib/gen/openapi/backend/types.gen.go -package backend openapi/backend/openapi.yaml
	oapi-codegen -config openapi/server.cfg.yaml -o lib/gen/openapi/backend/server.gen.go -package backend openapi/backend/openapi.yaml
	oapi-codegen -config openapi/types.cfg.yaml -o lib/gen/openapi/workflows/steps/web_feature_consumer/types.gen.go -package web_feature_consumer openapi/workflows/steps/web_feature_consumer/openapi.yaml
	oapi-codegen -config openapi/server.cfg.yaml -o lib/gen/openapi/workflows/steps/web_feature_consumer/server.gen.go -package web_feature_consumer openapi/workflows/steps/web_feature_consumer/openapi.yaml
	oapi-codegen -config openapi/types.cfg.yaml -o lib/gen/openapi/workflows/steps/common/repo_downloader/types.gen.go -package repo_downloader openapi/workflows/steps/common/repo_downloader/openapi.yaml
	oapi-codegen -config openapi/server.cfg.yaml -o lib/gen/openapi/workflows/steps/common/repo_downloader/server.gen.go -package repo_downloader openapi/workflows/steps/common/repo_downloader/openapi.yaml
	npx openapi-typescript openapi/backend/openapi.yaml -o lib/gen/openapi/ts-webstatus.dev-backend-types/types.d.ts

jsonschema:
	quicktype \
		--src jsonschema/web-platform-dx_web-features/defs.schema.json \
		--src-lang schema \
		--lang go \
		--top-level FeatureData \
		--out lib/gen/jsonschema/web_platform_dx__web_features/feature_data.go \
		--package web_platform_dx__web_features \
		--field-tags json

golint-version:
	golangci-lint --version

frontend-deps:
	npm install -w frontend 

lint: golint-version frontend-deps
	go list -f '{{.Dir}}/...' -m | xargs golangci-lint run
	npm run lint -w frontend
	terraform fmt -recursive -check .
	shellcheck .devcontainer/*.sh

lint-fix: frontend-deps
	npm run lint-fix -w frontend
	terraform fmt -recursive .

download-addlicense:
	go install github.com/google/addlicense@latest

license-check: download-addlicense
	addlicense -check $(ADDLICENSE_ARGS) .
license-fix: download-addlicense
	addlicense $(ADDLICENSE_ARGS) .
