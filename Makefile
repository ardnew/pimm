# ==============================================================================
#
#  ENVIRONMENT & PROJECT CONFIGURATION
#
# ==============================================================================
#

# -- static project definitions ------------------------------------------------

project    = pimmp
configpath = $(HOME)/.$(project)
importpath = ardnew.com/$(project)
gopathsrc  = $(GOPATH)/src
gopathbin  = $(GOPATH)/bin

dbgarg-verbosity = -verbose
dbgarg-climode = -cli

# -- define version info with version control ----------------------------------

version   = 0.1
branch    = $(shell git symbolic-ref --short HEAD)
revision  = $(shell git rev-parse --short HEAD)
buildtime = $(shell date -u '+%FT%TZ')

#
# something not working with the following, the subshell expansions come back as
# empty strings when being passed to printf for some reason...?
#
#$(shell \
#  if git rev-parse --is-inside-work-tree >/dev/null 2>&1 ; then \
#    printf "%s@%s" "$(git symbolic-ref --short -q HEAD)" "$(git rev-parse --short HEAD)" ; \
#  elif svn info >/dev/null 2>&1 ; then \
#    printf "r%s" "$(svn info | command grep -oP '^Revision:\s*\K\d+')" ; \
#  else \
#    printf "(unversioned)" ; \
#  fi \
#)

# -- other configuration for test/eval environment -----------------------------

# to construct lib paths in a recipe: $(addprefix $(mediaroot)/,$(mediadirs))
mediaroot = /mnt/media
mediadirs = movies tv
racereport = ./race-report/pid

# -- go flags (see: go help build) ---------------------------------------------

# run with e.g. `make install USER_GOFLAGS="-race"` to enable the race detector.
goflags-release =
goflags         = $(USER_GOFLAGS)

# -- compiler flags (see: go tool compile -help) -------------------------------

gcflags-release =
gcflags         = all='-N -l'

# -- linker flags (see: go tool link -help) ------------------------------------

ldflags-version = -X "main.identity=$(project)" -X "main.version=$(version)" -X "main.branch=$(branch)" -X "main.revision=$(revision)" -X "main.buildtime=$(buildtime)"
ldflags-release = '-w -s $(ldflags-version)'
ldflags         = '$(ldflags-version)'



# ==============================================================================
#
#  TARGET DEFINITION
#
# ==============================================================================
#

# -- janitorial / cleanup targets ----------------------------------------------

.PHONY: rinse clean scrub

rinse:
	rm -rf "$(configpath)"

clean:
	rm -f "$(gopathsrc)/$(importpath)/$(project)"
	rm -f "$(gopathbin)/$(project)"

scrub: rinse clean

# -- compilation targets -------------------------------------------------------

.PHONY: build install

build:
	go build $(goflags) -gcflags=$(gcflags) -ldflags=$(ldflags) "$(importpath)"

install:
	go install $(goflags) -gcflags=$(gcflags) -ldflags=$(ldflags) "$(importpath)"

# -- test / evaluation targets -------------------------------------------------

.PHONY: tui-single-lib tui-dual-lib cli-single-lib cli-dual-lib
.PHONY: race-tui-single-lib race-tui-dual-lib race-cli-single-lib race-cli-dual-lib
.PHONY: debug-tui-single-lib debug-tui-dual-lib debug-cli-single-lib debug-cli-dual-lib

tui-single-lib: install
	$(project) $(dbgarg-verbosity) $(mediaroot)

tui-dual-lib: install
	$(project) $(dbgarg-verbosity) $(addprefix $(mediaroot)/,$(mediadirs))

cli-single-lib: install
	$(project) $(dbgarg-verbosity) $(dbgarg-climode) $(mediaroot)

cli-dual-lib: install
	$(project) $(dbgarg-verbosity) $(dbgarg-climode) $(addprefix $(mediaroot)/,$(mediadirs))


race-tui-single-lib:
	make install USER_GOFLAGS="-race"
	GORACE="log_path=$(racereport) strip_path_prefix=$(gopathsrc)" $(project) $(dbgarg-verbosity) $(mediaroot)

race-tui-dual-lib:
	make install USER_GOFLAGS="-race"
	GORACE="log_path=$(racereport) strip_path_prefix=$(gopathsrc)" $(project) $(dbgarg-verbosity) $(addprefix $(mediaroot)/,$(mediadirs))

race-cli-single-lib:
	make install USER_GOFLAGS="-race"
	GORACE="log_path=$(racereport) strip_path_prefix=$(gopathsrc)" $(project) $(dbgarg-verbosity) $(dbgarg-climode) $(mediaroot)

race-cli-dual-lib:
	make install USER_GOFLAGS="-race"
	GORACE="log_path=$(racereport) strip_path_prefix=$(gopathsrc)" $(project) $(dbgarg-verbosity) $(dbgarg-climode) $(addprefix $(mediaroot)/,$(mediadirs))


debug-tui-single-lib: install
	dlv exec $(project) -- $(dbgarg-verbosity) $(mediaroot)

debug-tui-dual-lib: install
	dlv exec $(project) -- $(dbgarg-verbosity) $(addprefix $(mediaroot)/,$(mediadirs))

debug-cli-single-lib: install
	dlv exec $(project) -- $(dbgarg-verbosity) $(dbgarg-climode) $(mediaroot)

debug-cli-dual-lib: install
	dlv exec $(project) -- $(dbgarg-verbosity) $(dbgarg-climode) $(addprefix $(mediaroot)/,$(mediadirs))

# -- profiling targets ---------------------------------------------------------

# .PHONY: profile-single-lib-cpu profile-dual-lib-cpu profile-single-lib-mem profile-dual-lib-mem

# profile-single-lib-cpu: clean build
# 	go test -args -verbose $(mediaroot)
# 	go tool pprof --pdf ./$(project) ./cpu.pprof > cpu-prof.pdf

# profile-dual-lib-cpu: clean build
# 	go test -args -verbose $(addprefix $(mediaroot)/,$(mediadirs))
# 	go tool pprof --pdf ./$(project) ./cpu.pprof > cpu-prof.pdf

# profile-single-lib-mem: clean build
# 	go test -args -verbose $(mediaroot)
# 	go tool pprof --pdf ./$(project) ./cpu.pprof > cpu-prof.pdf

# profile-dual-lib-mem: clean build
# 	go test -args -verbose $(addprefix $(mediaroot)/,$(mediadirs))
# 	go tool pprof --pdf ./$(project) ./cpu.pprof > cpu-prof.pdf


