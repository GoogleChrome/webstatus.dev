module github.com/GoogleChrome/webstatus.dev/workflows/steps/services/bcd_consumer

go 1.22.0

replace github.com/GoogleChrome/webstatus.dev/lib => ../../../../lib

replace github.com/GoogleChrome/webstatus.dev/lib/gen => ../../../../lib/gen

require github.com/GoogleChrome/webstatus.dev/lib/gen v0.0.0-00010101000000-000000000000
