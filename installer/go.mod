module github.com/diegoclair/harness/installer

go 1.26.3

require (
	github.com/diegoclair/harness/pkg/release v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

// The shared package lives in this repo; resolve it locally so the module
// builds standalone (CI runs from this directory) as well as under go.work.
replace github.com/diegoclair/harness/pkg/release => ../pkg/release
