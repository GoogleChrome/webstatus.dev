module github.com/GoogleChrome/webstatus.dev/workers/chime

go 1.25.4

replace github.com/GoogleChrome/webstatus.dev/lib => ../../lib

replace github.com/GoogleChrome/webstatus.dev/lib/gen => ../../lib/gen

require golang.org/x/oauth2 v0.33.0

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/google/uuid v1.6.0
	golang.org/x/sys v0.38.0 // indirect
)
