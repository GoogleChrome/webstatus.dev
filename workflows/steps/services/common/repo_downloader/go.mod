module github.com/GoogleChrome/webstatus.dev/workflows/steps/services/common/repo_downloader

go 1.22.0

require (
	cloud.google.com/go/storage v1.42.0
	github.com/GoogleChrome/webstatus.dev/lib v0.0.0-20240617233854-afc779445ed9
	github.com/GoogleChrome/webstatus.dev/lib/gen v0.0.0-00010101000000-000000000000
	github.com/google/go-github/v55 v55.0.0
)

require (
	cloud.google.com/go/auth v0.5.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.2 // indirect
	github.com/ProtonMail/go-crypto v1.0.0 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/cloudflare/circl v1.3.9 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/getkin/kin-openapi v0.125.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.8 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/invopop/yaml v0.2.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/oapi-codegen/runtime v1.1.1 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.52.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.52.0 // indirect
	go.opentelemetry.io/otel v1.27.0 // indirect
	go.opentelemetry.io/otel/metric v1.27.0 // indirect
	go.opentelemetry.io/otel/trace v1.27.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	cloud.google.com/go v0.115.0 // indirect
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	cloud.google.com/go/iam v1.1.8 // indirect
	github.com/go-chi/chi/v5 v5.0.14
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/api v0.185.0
	google.golang.org/genproto v0.0.0-20240617180043-68d350f18fd4 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240617180043-68d350f18fd4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240617180043-68d350f18fd4 // indirect
	google.golang.org/grpc v1.64.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/GoogleChrome/webstatus.dev/lib => ../../../../../lib

replace github.com/GoogleChrome/webstatus.dev/lib/gen => ../../../../../lib/gen
