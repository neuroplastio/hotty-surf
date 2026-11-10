# hotty-surf. Toolchain from mise.toml (Go, TinyGo): `mise install` first.
# Every step runs under CAP (below): GO and GOFMT carry it, GO_BARE is go
# for a step that CAP already wraps.

# A machine's own settings (CAP, IDLE_CPUS, HOTTY, HOTTY_BLITZ, HOTTY_FONTS)
# go in $XDG_CONFIG_HOME/hotty-surf/make.mk, read here if it is there,
# before the defaults below, which suit CI and a first checkout. A variable
# on the command line wins over both (`make check CAP=`).
-include $(or $(XDG_CONFIG_HOME),$(HOME)/.config)/hotty-surf/make.mk

GO_BARE     ?= mise x -- go
GO          ?= $(CAP) $(GO_BARE)
GOFMT       ?= $(CAP) mise x -- gofmt
# staticcheck 2026.2.1, the first that knows Go 1.26.
STATICCHECK ?= honnef.co/go/tools/cmd/staticcheck@v0.8.1

.PHONY: check fmt tidy gen vet lint test bench print-cap clean

check: fmt tidy gen vet lint test look-test wasm stack gate legible   ## the gate

.PHONY: look-test
look-test:   ## the look loop passes each producer its theme names
	$(CAP) $(NODE) node --test scripts/look/look.test.cjs

fmt:   ## fails, listing them, when gofmt would change files
	@out="$$($(GOFMT) -l $$(git ls-files -co --exclude-standard '*.go'))"; \
	if [ -n "$$out" ]; then echo "gofmt would change:"; echo "$$out"; exit 1; fi

tidy:   ## fails when go.mod or go.sum is not what go mod tidy makes
	$(GO) mod tidy -diff

gen:    ## fails when a generated file is not what its go:generate makes (surfgen)
	@t=$$(mktemp); trap 'rm -f "$$t"' EXIT; \
	sha256sum $$(git ls-files -co --exclude-standard '*_gen.go') >"$$t"; \
	$(GO) generate ./... || exit 1; \
	out="$$(sha256sum --quiet -c "$$t" 2>/dev/null | sed 's/: FAILED$$//')"; \
	if [ -n "$$out" ]; then echo "go generate changed:"; echo "$$out"; exit 1; fi

vet:
	$(GO) vet ./...

lint:   ## staticcheck, pinned; from the module cache when it has it
	GOPROXY=file://$(HOME)/go/pkg/mod/cache/download,https://proxy.golang.org $(GO) run $(STATICCHECK) ./...

test:   ## the tests, with the race detector
	$(GO) test -race ./...

# bench times the machine: run it while nothing else runs, at normal
# priority (`make bench IDLE=` where a CAP holds steps at idle), as
# vault/knowledge/measures.md says. BENCHFLAGS:
# e.g. -count=6, or -bench Update.
BENCHFLAGS ?=

bench:   ## the benchmarks, which check doesn't run: they time the machine
	$(GO) test -run '^$$' -bench . -benchmem $(BENCHFLAGS) ./pkg/conformance ./internal/screens

print-cap:   ## the CAP the steps run under, for a script that builds outside make
	@echo '$(CAP)'

clean:
	rm -rf bin .shots

# The framework under WebAssembly: its packages (FRAMEWORK) built with a
# 32-bit int, as TinyGo's wasm has it, and built and tested under Go's
# js/wasm, in Node, and TinyGo's wasm target; and examples/wasm, a
# component run through the runtime on a host in the process, run under
# both. examples/wasm-surf, the same component drawn by a .surf template,
# is built and run in each mode under Go's js/wasm here, and under TinyGo
# by wasm-sizes. vault/knowledge/wasm.md says what runs where;
# vault/knowledge/measures.md has the sizes (wasm-sizes).
#
# FRAMEWORK is the framework's packages: a new package of the framework's
# goes in it, with a test (tinygo test builds a package with no test file
# not at all, so make wasm fails on one). cmd, screens, gallery and scripts
# are not the framework.
FRAMEWORK := . ./pkg/markup ./pkg/mdi ./pkg/conformance ./surfui ./surfrun ./pkg/cellblock ./pkg/cells ./surflive ./surftest ./internal/stories ./surflua ./internal/stackprobe
# TINYGO is the TinyGo that mise.toml pins, as mise has it: `mise
# install` installs it. A TinyGo build takes several GB, and other work
# on the machine runs TinyGo too, so its packages build and test one at a
# time.
ifeq ($(origin TINYGO),undefined)
TINYGO    := $(shell mise which tinygo 2>/dev/null)
endif
# CAP wraps every step: go, gofmt, TinyGo, Node and the gate. It is empty
# here. A TinyGo wasm test that runs away, or a generator in a loop, can
# take tens of GB, so a machine shared with other jobs caps each step's
# memory, and a runaway kills only itself:
#
#   CAP = systemd-run --user --scope -q -p MemoryMax=12G -p MemorySwapMax=2G $(IDLE) --
#
# An exit of 137 is then the cap's kill: find what grew (a loop, a
# recursion on TinyGo's 64 KB stacks), and don't raise the cap. IDLE, in
# such a CAP, keeps the steps on the CPUs IDLE_CPUS names, at idle
# priority, out of the way of timed work: `make bench IDLE=` runs at
# normal priority on every core.
IDLE_CPUS ?=
IDLE      ?= $(if $(IDLE_CPUS),-p CPUWeight=idle -p AllowedCPUs=$(IDLE_CPUS))
CAP       ?=
# PIN keeps a step that starts a browser on IDLE_CPUS too: Chromium moves
# each of its processes into a scope of its own, out of CAP's AllowedCPUs,
# and a CPU affinity survives that move where a cpuset does not. It is
# empty with no CAP or no IDLE.
PIN       ?= $(if $(and $(CAP),$(IDLE)),taskset -c $(IDLE_CPUS))
WASM_OUT  := bin/wasm
# What runs a js/wasm module in Node: Go's (for a program, or a package's
# tests), and TinyGo's, as tinygo test runs its own.
GO_WASM_EXEC = "$$($(GO_BARE) env GOROOT)/lib/wasm/go_js_wasm_exec"
TINYGO_EXEC  = node "$$($(TINYGO) env TINYGOROOT)/targets/wasm_exec_node.js"

.PHONY: wasm wasm-tinygo wasm-sizes stack

wasm: wasm-tinygo   ## the framework built with a 32-bit int, and built and tested under Go's js/wasm and TinyGo's; examples/wasm run under both
	@bare="$$($(GO_BARE) list -f '{{if not (or .TestGoFiles .XTestGoFiles)}}{{.ImportPath}}{{end}}' $(FRAMEWORK))"; \
	if [ -n "$$bare" ]; then echo "no test file in $$bare: tinygo test would not build it, and each package of FRAMEWORK needs one"; exit 1; fi
	GOARCH=386 $(GO) build $(FRAMEWORK) ./examples/wasm ./examples/wasm-surf
	GOARCH=386 $(GO) vet $(FRAMEWORK) ./examples/wasm ./examples/wasm-surf
	GOOS=js GOARCH=wasm $(GO) vet $(FRAMEWORK) ./examples/wasm ./examples/wasm-surf
	GOOS=js GOARCH=wasm $(GO) vet -tags compiled ./examples/wasm-surf
	GOOS=js GOARCH=wasm $(GO) build $(FRAMEWORK)
	GOOS=js GOARCH=wasm $(CAP) $(NODE) $(GO_BARE) test -exec $(GO_WASM_EXEC) $(FRAMEWORK)
	GOOS=js GOARCH=wasm $(GO) build -o $(WASM_OUT)/go.wasm ./examples/wasm
	$(CAP) $(NODE) $(GO_WASM_EXEC) $(WASM_OUT)/go.wasm
	GOOS=js GOARCH=wasm $(GO) build -o $(WASM_OUT)/go-surf.wasm ./examples/wasm-surf
	$(CAP) $(NODE) $(GO_WASM_EXEC) $(WASM_OUT)/go-surf.wasm
	GOOS=js GOARCH=wasm $(GO) build -tags compiled -o $(WASM_OUT)/go-surf-compiled.wasm ./examples/wasm-surf
	$(CAP) $(NODE) $(GO_WASM_EXEC) $(WASM_OUT)/go-surf-compiled.wasm
	for p in $(FRAMEWORK); do $(CAP) $(NODE) $(TINYGO) test -target=wasm $$p || exit 1; done
	$(CAP) $(NODE) $(TINYGO) build -target=wasm -o $(WASM_OUT)/tinygo.wasm ./examples/wasm
	$(CAP) $(NODE) $(TINYGO_EXEC) $(WASM_OUT)/tinygo.wasm

# TinyGo runs a goroutine on a fixed stack, 64 KB on wasm, whose overflow
# corrupts the heap silently: its canary is checked only at a pause. A
# render nests at most surf.MaxDepth levels, and a file its elements
# markup.MaxDepth deep, which keeps a render, a Load and the walks of a
# render's tree within half of it (vault/knowledge/wasm.md, "Stack").
# make stack holds them to it. It vets the probes' files with Go first
# (the paint reads the stack by its address); measures under TinyGo, on
# 1 MB stacks, what each path needs at the limit, by probes that do
# nothing without the surfstack tag (internal/stackprobe), and fails
# where one is past its budget (pkg/conformance/stack_test.go); then renders
# at the limit and past it, and walks the tree, under TinyGo's own 64 KB
# stack (TestTooDeep, cases/deep.txt and toodeep.txt). It counts bytes,
# not time, so check runs it.
stack: wasm-tinygo   ## the stack a render, a Load and a render's tree's walks need under TinyGo's wasm at surf.MaxDepth, against the budget; then the limit's cases on TinyGo's 64 KB stack
	GOOS=js GOARCH=wasm $(GO) vet -tags surfstack,tinygo -unsafeptr=false ./internal/stackprobe ./pkg/conformance ./surfrun ./surflua
	for p in ./pkg/conformance ./surfrun ./surflua; do $(CAP) $(NODE) $(TINYGO) test -tags surfstack -target=wasm -stack-size=1MB -run '^TestStackUse$$' -v $$p || exit 1; done
	$(CAP) $(NODE) $(TINYGO) test -target=wasm -run '^(TestTooDeep|TestCases)$$/^(deep|toodeep)$$' -v ./pkg/conformance

wasm-sizes: wasm-tinygo   ## examples/wasm's and examples/wasm-surf's modules by each toolchain and its flags, each run: vault/knowledge/measures.md's tables (sizes don't time the machine)
	@GO="$(GO_BARE)" NODE="$(NODE)" TINYGO="$(TINYGO)" CAP="$(CAP)" bash scripts/wasm-sizes.sh

wasm-tinygo:
	@test -x "$(TINYGO)" || { echo "no TinyGo at '$(TINYGO)': run mise install, or set TINYGO=<path to tinygo>"; mise which tinygo; exit 1; }

# F5's fidelity gate (scripts/gate): every story in hotty-blitz and in
# Chromium, diffed, and judged against the known failures, the Blitz gaps
# (GATE_BASELINE; scripts/gate/gate.cjs says how). It needs a hotty-blitz
# release build, its fonts from HOTTY_FONTS (a checkout next to this one or
# HOTTY_BLITZ has them), and Node for Playwright. GATE_FLAGS: e.g. --only
# button-states, or --inject <css>.
#
# HOTTY is the hotty-blitz the hosts ship: the baseline is written with
# v0.7.0, hottyterm's since 26.11.06, so build that tag (make blitz and make
# release in a hotty-blitz checkout). HOTTY_BLITZ is the checkout, next to
# this one by default. HOTTY_FONTS is the fonts/ Chromium loads: AdwaitaMono
# and IBM Plex Sans, which the kit draws Compact and Roomy in, the same
# faces hotty-blitz draws with (it ships Plex from 0.6.9), so both sides are
# machine-independent.
HOTTY_BLITZ   ?= $(abspath $(patsubst %/Cargo.toml,%,$(firstword $(wildcard ../hotty-blitz/Cargo.toml))))
HOTTY         ?= $(HOTTY_BLITZ)/target/release/hotty
HOTTY_FONTS   ?= $(HOTTY_BLITZ)/crates/hotty-blitz/fonts
NODE          ?= mise x node@26.10.0 --
GATE_BASELINE ?= scripts/gate/baseline.tsv
GATE_FLAGS    ?=

.PHONY: gate gate-baseline

gate: scripts/gate/node_modules   ## F5: every story in Blitz and Chromium, diffed, against the known failures: .shots/gate/report.md
	@test -x "$(HOTTY)" || { echo "no hotty at $(HOTTY): run make release in hotty-blitz, or set HOTTY_BLITZ"; exit 1; }
	$(CAP) $(PIN) $(NODE) node --test scripts/gate/font-scale.test.cjs
	$(GO) build -o bin/stories ./internal/cmd/stories
	$(CAP) $(PIN) $(NODE) node scripts/gate/gate.cjs --hotty $(HOTTY) --fonts $(HOTTY_FONTS) \
	  --stories bin/stories --out .shots/gate $(if $(GATE_BASELINE),--baseline $(GATE_BASELINE)) $(GATE_FLAGS)

gate-baseline: scripts/gate/node_modules   ## writes the known failures again: when a Blitz fix lands, or a gap is listed
	@test -x "$(HOTTY)" || { echo "no hotty at $(HOTTY): run make release in hotty-blitz, or set HOTTY_BLITZ"; exit 1; }
	$(GO) build -o bin/stories ./internal/cmd/stories
	-$(CAP) $(PIN) $(NODE) node scripts/gate/gate.cjs --hotty $(HOTTY) --fonts $(HOTTY_FONTS) \
	  --stories bin/stories --out .shots/gate --write-baseline $(GATE_BASELINE)

# The legible gate (scripts/gate/legible.cjs says how): every story the
# gallery lists, drawn as the gallery draws it at the story's own width,
# in both modes and every density, and measured in Chromium. It fails
# on what runs past the right edge of its box, and on a text its
# ellipsis cuts to fewer than 4 characters. F5's gate looks at
# stories.All's stories alone, each on its own, and passed while the
# gallery drew the palette in 20 cells. LEGIBLE_KNOWN is the stories that
# fail today, each with its reason: to be empty. LEGIBLE_FLAGS: e.g.
# --only palette, or --width 20 with LEGIBLE_KNOWN= (the calibration).
LEGIBLE_KNOWN ?= scripts/gate/legible-known.tsv
LEGIBLE_FLAGS ?=

.PHONY: legible

legible: scripts/gate/node_modules   ## the gallery's stories at their widths, measured: nothing past its box, no text cut to under 4 characters: .shots/legible/report.md
	@test -x "$(HOTTY)" || { echo "no hotty at $(HOTTY): run make release in hotty-blitz, or set HOTTY_BLITZ"; exit 1; }
	$(GO) build -o bin/stories ./internal/cmd/stories
	$(CAP) $(PIN) $(NODE) node scripts/gate/legible.cjs --hotty $(HOTTY) --fonts $(HOTTY_FONTS) \
	  --stories bin/stories --out .shots/legible $(if $(LEGIBLE_KNOWN),--known $(LEGIBLE_KNOWN)) $(LEGIBLE_FLAGS)

# The look loop (scripts/look/README.md): a story drawn in Chromium and in
# Blitz, a board of the design (or one element of it) drawn in Chromium
# with its kit.css, and the sheets that compare them, in .shots/look/.
# LOOK is what to run, once for each of LOOK_THEMES; by default the
# button's states against the Components board's button row. DESIGN is
# the design's directory, ../design by default:
#
#   make look LOOK='run list/selection --board Components-dark.html --select "section:nth-of-type(8)"'
#   make look LOOK='story button/states --scale 2' LOOK_THEMES=light
LOOK        ?= run button/states --board Components-dark.html --select 'section:nth-of-type(4) > div:nth-child(2)'
LOOK_THEMES ?= dark light
DESIGN      ?=

.PHONY: look

look: scripts/gate/node_modules   ## edit, render, compare: a story in Chromium and Blitz beside a board of the design, .shots/look/
	@test -x "$(HOTTY)" || { echo "no hotty at $(HOTTY): run make release in hotty-blitz, or set HOTTY_BLITZ"; exit 1; }
	$(GO) build -o bin/stories ./internal/cmd/stories
	for t in $(LOOK_THEMES); do $(CAP) $(PIN) $(NODE) node scripts/look/look.cjs $(LOOK) --theme $$t --hotty $(HOTTY) \
	  --fonts $(HOTTY_BLITZ)/crates/hotty-blitz/fonts --stories bin/stories $(if $(DESIGN),--design $(DESIGN)) || exit 1; done

# The pinned Playwright, and the Chromium it drives.
scripts/gate/node_modules: scripts/gate/package.json scripts/gate/package-lock.json
	cd scripts/gate && $(CAP) $(NODE) npm ci --no-audit --no-fund && $(CAP) $(NODE) npx playwright install chromium-headless-shell
	@touch $@
