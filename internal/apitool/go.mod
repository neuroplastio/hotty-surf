// apitool: hotty-surf's API lock and review packs. A module of its own, so
// that the framework's go.mod doesn't carry its dependencies.
module neuroplast.io/hotty-surf/internal/apitool

go 1.26.0

require (
	github.com/yuin/goldmark v1.8.6
	golang.org/x/tools v0.52.0
)

require (
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
)
