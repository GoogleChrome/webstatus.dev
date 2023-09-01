
gen: openapi jsonschema

openapi:
	oapi-codegen -config openapi/types.cfg.yaml -o lib/gen/openapi/backend/types.gen.go -package backend openapi/backend/openapi.yaml
	oapi-codegen -config openapi/server.cfg.yaml -o lib/gen/openapi/backend/server.gen.go -package backend openapi/backend/openapi.yaml
	oapi-codegen -config openapi/types.cfg.yaml -o lib/gen/openapi/workflows/steps/web_feature_consumer/types.gen.go -package web_feature_consumer openapi/workflows/steps/web_feature_consumer/openapi.yaml
	oapi-codegen -config openapi/server.cfg.yaml -o lib/gen/openapi/workflows/steps/web_feature_consumer/server.gen.go -package web_feature_consumer openapi/workflows/steps/web_feature_consumer/openapi.yaml
	oapi-codegen -config openapi/types.cfg.yaml -o lib/gen/openapi/workflows/steps/common/repo_downloader/types.gen.go -package repo_downloader openapi/workflows/steps/common/repo_downloader/openapi.yaml
	oapi-codegen -config openapi/server.cfg.yaml -o lib/gen/openapi/workflows/steps/common/repo_downloader/server.gen.go -package repo_downloader openapi/workflows/steps/common/repo_downloader/openapi.yaml

jsonschema:
	quicktype \
		--src jsonschema/web-platform-dx_web-features/defs.schema.json \
		--src-lang schema \
		--lang go \
		--top-level FeatureData \
		--out lib/gen/jsonschema/web_platform_dx__web_features/feature_data.go \
		--package web_platform_dx__web_features \
		--field-tags json

lint-version:
	golangci-lint --version

lint: lint-version
	go list -f '{{.Dir}}/...' -m | xargs golangci-lint run