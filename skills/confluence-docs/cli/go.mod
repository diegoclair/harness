module github.com/diegoclair/harness/skills/confluence-docs/cli

go 1.26.3

require (
	github.com/diegoclair/harness/pkg/atlassian v0.0.0-00010101000000-000000000000
	github.com/diegoclair/harness/pkg/release v0.0.0-00010101000000-000000000000
)

require github.com/yuin/goldmark v1.7.8 // indirect

replace github.com/diegoclair/harness/pkg/atlassian => ../../../pkg/atlassian

replace github.com/diegoclair/harness/pkg/release => ../../../pkg/release
