# hotty-surf. Toolchain from mise.toml (Go, TinyGo, templ, msgp): `mise
# install` first. Every step runs under CAP (below): GO and GOFMT carry it,
# GO_BARE is go for a step that CAP already wraps.

# A machine's own settings (CAP, IDLE_CPUS) go in
# $XDG_CONFIG_HOME/hotty-surf/make.mk, read here if it is there, before the
# defaults below, which suit CI and a first checkout. A variable on the
# command line wins over both (`make check CAP=`).
-include $(or $(XDG_CONFIG_HOME),$(HOME)/.config)/hotty-surf/make.mk

GO_BARE     ?= mise x -- go
GO          ?= $(CAP) $(GO_BARE)
GOFMT       ?= $(CAP) mise x -- gofmt
# staticcheck 2026.2.1, the first that knows Go 1.26.
STATICCHECK ?= honnef.co/go/tools/cmd/staticcheck@v0.8.1

# CAP wraps every step. It is empty here. A machine shared with other jobs
# caps each step's memory, so that a runaway kills only itself:
#
#   CAP = systemd-run --user --scope -q -p MemoryMax=12G -p MemorySwapMax=2G $(IDLE) --
#
# IDLE, in such a CAP, keeps the steps on the CPUs IDLE_CPUS names, at idle
# priority.
IDLE_CPUS ?=
IDLE      ?= $(if $(IDLE_CPUS),-p CPUWeight=idle -p AllowedCPUs=$(IDLE_CPUS))
CAP       ?=

# SUBMODULES are the modules of their own inside this repository, each with
# a go.mod: surftea takes Bubble Tea, which the framework does not, and
# internal/apitool the API lock's tools.
SUBMODULES := surftea internal/apitool

.PHONY: check fmt tidy gen vet lint test canary tinygo api api-check api-frozen api-approve pack coverage print-cap clean

check: fmt tidy gen vet lint test tinygo api-check coverage   ## the gate

fmt:   ## fails, listing them, when gofmt would change files
	@out="$$($(GOFMT) -l $$(git ls-files -co --exclude-standard '*.go'))"; \
	if [ -n "$$out" ]; then echo "gofmt would change:"; echo "$$out"; exit 1; fi

tidy:   ## fails when go.mod or go.sum is not what go mod tidy makes
	$(GO) mod tidy -diff
	@for m in $(SUBMODULES); do $(GO) -C $$m mod tidy -diff || exit 1; done

gen:    ## fails when a generated file (templ's, msgp's) is not what its source makes
	@t=$$(mktemp); trap 'rm -f "$$t"' EXIT; \
	files="$$(git ls-files -co --exclude-standard '*_templ.go' '*_gen.go')"; \
	sha256sum $$files >"$$t"; \
	for f in $$(git ls-files -co --exclude-standard '*.templ'); do \
	  (cd "$$(dirname "$$f")" && $(CAP) mise x -- templ generate -f "$$(basename "$$f")" >/dev/null 2>&1) || exit 1; done; \
	$(GO) generate ./... || exit 1; \
	out="$$(sha256sum --quiet -c "$$t" 2>/dev/null | sed 's/: FAILED$$//')"; \
	if [ -n "$$out" ]; then echo "generating changed:"; echo "$$out"; exit 1; fi

# The surfdev build (live reload's: register_dev.go, reload/dev.go,
# run/dev.go, internal/devchild) is vetted, linted and tested too.
vet:
	$(GO) vet ./...
	$(GO) vet -tags surfdev ./...
	GO="$(GO)" sh scripts/templmod-check.sh vet
	@for m in $(SUBMODULES); do $(GO) -C $$m vet ./... || exit 1; done

lint:   ## staticcheck, pinned; from the module cache when it has it
	GOPROXY=file://$(HOME)/go/pkg/mod/cache/download,https://proxy.golang.org $(GO) run $(STATICCHECK) ./...
	GOPROXY=file://$(HOME)/go/pkg/mod/cache/download,https://proxy.golang.org $(GO) run $(STATICCHECK) -tags surfdev ./...
	GOPROXY=file://$(HOME)/go/pkg/mod/cache/download,https://proxy.golang.org GO="$(GO)" STATICCHECK=$(STATICCHECK) sh scripts/templmod-check.sh lint
	@for m in $(SUBMODULES); do (cd $$m && GOPROXY=file://$(HOME)/go/pkg/mod/cache/download,https://proxy.golang.org $(GO) run $(STATICCHECK) ./...) || exit 1; done

test:   ## the tests, with the race detector
	$(GO) test -race ./...
	$(GO) test -race -tags surfdev ./internal/devchild/...
	@for m in $(SUBMODULES); do $(GO) -C $$m test -race ./... || exit 1; done

canary:   ## mods/ applied to each upstream's latest, and its tests (gov P-7; weekly in CI)
	GO="$(GO)" sh scripts/canary.sh

# The examples' release wasm, as TinyGo builds it (gov NEIO-15).
# scripts/wasm-stream.sh holds its stream to Go's, with node.
TINYGO ?= $(CAP) mise x -- tinygo
tinygo:   ## the examples build for js/wasm under TinyGo
	@mkdir -p bin
	@for e in tasks nav hello; do $(TINYGO) build -no-debug -target wasm -o bin/$$e.wasm ./examples/$$e || exit 1; done

# The API lock (NEIO-15's review loop). api/v0.7.txt is the approved API, in
# Go's api/ format; it changes only when the maintainer approves a pack.
# api/next.txt is the exported API as it is now, generated. What next has
# that the approved file hasn't is what the next pack reviews, each name
# with its reason in api/reasons.md.
#
#   make api           regenerates api/next.txt after an exported change
#   make api-check     fails when the exported API is not api/next.txt: no
#                      exported change goes in without showing in a pack
#   make api-frozen    fails while anything is not approved (CI on main)
#   make api-approve   folds next into the approved file: on the
#                      maintainer's approval only, named in the commit
#   make pack          the review pack, an HTML page (PACK)
APITOOL  = $(GO) -C internal/apitool run .
API_DIRS = ../.. ../../surftea
API_OK   = api/v0.7.txt
PACK    ?= $(or $(XDG_CACHE_HOME),$(HOME)/.cache)/hotty-surf/pack.html
PACK_TITLE ?= hotty-surf API pack
PACK_NOTES ?= api/notes.md
PACK_APPENDIX ?=

api:
	$(APITOOL) gen -o ../../api/next.txt $(API_DIRS)

api-check:
	@t=$$(mktemp); trap 'rm -f "$$t"' EXIT; \
	$(APITOOL) gen -o "$$t" $(API_DIRS) || exit 1; \
	if ! diff -u api/next.txt "$$t"; then \
	  echo "the exported API changed: make api, give each new name its reason in api/reasons.md, and put it in the next pack"; exit 1; fi

api-frozen: api-check
	$(APITOOL) diff -approved ../../$(API_OK) -next ../../api/next.txt

api-approve:
	{ grep '^#' $(API_OK); cat api/next.txt; } >$(API_OK).new && mv $(API_OK).new $(API_OK)

pack: api-check
	$(APITOOL) pack -approved ../../$(API_OK) -next ../../api/next.txt \
	  -reasons ../../api/reasons.md -title "$(PACK_TITLE)" -o "$(PACK)" \
	  $(if $(PACK_NOTES),-notes ../../$(PACK_NOTES)) \
	  $(foreach a,$(PACK_APPENDIX),-appendix $(abspath $(a))) \
	  $(foreach f,$(filter-out %_gen.go %_templ.go,$(wildcard examples/hello/*.go examples/hello/*.templ)),-file $(f)) \
	  $(API_DIRS)
	@echo "pack: $(PACK)"

# docs/coverage.md is written from ui/coverage.tsv, Basecoat's components
# one by one; it fails when the page is not what the table makes.
coverage:
	@t=$$(mktemp); trap 'rm -f "$$t"' EXIT; \
	sh scripts/coverage-md.sh "$$t" || exit 1; \
	if ! diff -q docs/coverage.md "$$t" >/dev/null; then \
	  echo "docs/coverage.md is not what ui/coverage.tsv makes: sh scripts/coverage-md.sh"; exit 1; fi

print-cap:   ## the CAP the steps run under, for a script that builds outside make
	@echo '$(CAP)'

clean:
	rm -rf bin .shots
