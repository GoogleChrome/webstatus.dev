module github.com/GoogleChrome/webstatus.dev/workflows/steps/services/uma_export

go 1.23.0

replace github.com/GoogleChrome/webstatus.dev/lib => ../../../../lib

replace github.com/GoogleChrome/webstatus.dev/lib/gen => ../../../../lib/gen

require (
	cloud.google.com/go v0.115.0
	github.com/GoogleChrome/webstatus.dev/lib v0.0.0-00010101000000-000000000000
)
