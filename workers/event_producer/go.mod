module github.com/GoogleChrome/webstatus.dev/workers/event_producer

go 1.25.4

replace github.com/GoogleChrome/webstatus.dev/lib => ../../lib

replace github.com/GoogleChrome/webstatus.dev/lib/gen => ../../lib/gen

require (
	github.com/GoogleChrome/webstatus.dev/lib v0.0.0-00010101000000-000000000000
	github.com/GoogleChrome/webstatus.dev/lib/gen v0.0.0-20251119220853-b545639c35ae
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/getkin/kin-openapi v0.133.0 // indirect
	github.com/go-openapi/jsonpointer v0.22.3 // indirect
	github.com/go-openapi/swag/jsonname v0.25.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.9.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/oapi-codegen/runtime v1.1.2 // indirect
	github.com/oasdiff/yaml v0.0.0-20250309154309-f31be36b4037 // indirect
	github.com/oasdiff/yaml3 v0.0.0-20250309153720-d2182401db90 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/woodsbury/decimal128 v1.4.0 // indirect
)
