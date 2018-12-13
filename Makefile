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
revision  = r$(shell svn info| \grep -oP '^Revision:\s*\K\d+')
buildtime = $(shell date -u '+%FT%TZ')

# -- go flags (see: go help build) ---------------------------------------------

goflags-release =
#goflags         = -race
goflags         =

# -- compiler flags (see: go tool compile -help) -------------------------------

gcflags-release =
gcflags         = all='-N -l'

# -- linker flags (see: go tool link -help) ------------------------------------

ldflags-version = -X "main.identity=$(project)" -X "main.version=$(version)" -X "main.revision=$(revision)" -X "main.buildtime=$(buildtime)"
ldflags-release = '-w -s $(ldflags-version)'
ldflags         = '$(ldflags-version)'



# ==============================================================================
#
#  TARGET DEFINITION
#
# ==============================================================================
#

# -- janitorial / cleanup targets ----------------------------------------------

.PHONY: rinse clean scrub sync-ripper-push sync-ripper-pull

rinse:
	rm -rf "$(configpath)"

clean:
	rm -f "$(gopathsrc)/$(importpath)/$(project)"
	rm -f "$(gopathbin)/$(project)"

scrub: rinse clean

sync-ripper-push:
	rsync -rave 'ssh -p 2222 -l andrew' $(gopathsrc)/$(importpath)/ ardnew.com:$(shell ssh ripper 'echo $$GOPATH/src | sed -E "s|^$$HOME|~|"')/$(importpath)

sync-ripper-pull:
	rsync -rave 'ssh -p 2222 -l andrew' ardnew.com:$(shell ssh ripper 'echo $$GOPATH/src | sed -E "s|^$$HOME|~|"')/$(importpath)/ $(gopathsrc)/$(importpath)

# -- compilation targets -------------------------------------------------------

.PHONY: build install

build:
	go build $(goflags) -gcflags=$(gcflags) -ldflags=$(ldflags) "$(importpath)"

install:
	go install $(goflags) -gcflags=$(gcflags) -ldflags=$(ldflags) "$(importpath)"

# -- test / evaluation targets -------------------------------------------------

.PHONY: tui-single-lib tui-dual-lib cli-single-lib cli-dual-lib
.PHONY: debug-tui-single-lib debug-tui-dual-lib debug-cli-single-lib debug-cli-dual-lib

tui-single-lib: install
	$(project) $(dbgarg-verbosity) /mnt/SG4TB-NIX

tui-dual-lib: install
	$(project) $(dbgarg-verbosity) /mnt/SG4TB-NIX/movies /mnt/SG4TB-NIX/tv

cli-single-lib: install
	$(project) $(dbgarg-verbosity) $(dbgarg-climode) /mnt/SG4TB-NIX

cli-dual-lib: install
	$(project) $(dbgarg-verbosity) $(dbgarg-climode) /mnt/SG4TB-NIX/movies /mnt/SG4TB-NIX/tv

debug-tui-single-lib: install
	dlv exec $(project) -- $(dbgarg-verbosity) /mnt/SG4TB-NIX

debug-tui-dual-lib: install
	dlv exec $(project) -- $(dbgarg-verbosity) /mnt/SG4TB-NIX/movies /mnt/SG4TB-NIX/tv

debug-cli-single-lib: install
	dlv exec $(project) -- $(dbgarg-verbosity) $(dbgarg-climode) /mnt/SG4TB-NIX

debug-cli-dual-lib: install
	dlv exec $(project) -- $(dbgarg-verbosity) $(dbgarg-climode) /mnt/SG4TB-NIX/movies /mnt/SG4TB-NIX/tv

# -- profiling targets ---------------------------------------------------------

# .PHONY: profile-single-lib-cpu profile-dual-lib-cpu profile-single-lib-mem profile-dual-lib-mem

# profile-single-lib-cpu: clean build
# 	go test -args -verbose /mnt/SG4TB-NIX
# 	go tool pprof --pdf ./$(project) ./cpu.pprof > cpu-prof.pdf

# profile-dual-lib-cpu: clean build
# 	go test -args -verbose /mnt/SG4TB-NIX/movies /mnt/SG4TB-NIX/tv
# 	go tool pprof --pdf ./$(project) ./cpu.pprof > cpu-prof.pdf

# profile-single-lib-mem: clean build
# 	go test -args -verbose /mnt/SG4TB-NIX
# 	go tool pprof --pdf ./$(project) ./cpu.pprof > cpu-prof.pdf

# profile-dual-lib-mem: clean build
# 	go test -args -verbose /mnt/SG4TB-NIX/movies /mnt/SG4TB-NIX/tv
# 	go tool pprof --pdf ./$(project) ./cpu.pprof > cpu-prof.pdf