module github.com/GoogleChrome/webstatus.dev/workflows/steps/services/uma_export

go 1.23.0

replace github.com/GoogleChrome/webstatus.dev/lib => ../../../../lib

replace github.com/GoogleChrome/webstatus.dev/lib/gen => ../../../../lib/gen

require (
	github.com/GoogleChrome/webstatus.dev/lib v0.0.0-20240819152439-9f4742e5167d
	golang.org/x/oauth2 v0.22.0
)

require (
	cel.dev/expr v0.15.0 // indirect
	cloud.google.com/go v0.115.0 // indirect
	cloud.google.com/go/auth v0.8.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.4 // indirect
	cloud.google.com/go/spanner v1.66.0 // indirect
	github.com/GoogleChrome/webstatus.dev/lib/gen v0.0.0-00010101000000-000000000000 // indirect
	github.com/GoogleCloudPlatform/grpc-gcp-go/grpcgcp v1.5.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20240723142845-024c85f92f20 // indirect
	github.com/envoyproxy/go-control-plane v0.12.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.1.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.13.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/exp v0.0.0-20240808152545-0cdaa3abc0fa // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/time v0.6.0 // indirect
	golang.org/x/xerrors v0.0.0-20240716161551-93cc26a95ae9 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240808171019-573a1156607a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240808171019-573a1156607a // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

require (
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	google.golang.org/api v0.192.0
)
