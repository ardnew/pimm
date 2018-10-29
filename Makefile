# ==============================================================================
#
#  ENVIRONMENT & PROJECT CONFIGURATION
#
# ==============================================================================
#

# -- static project definitions ------------------------------------------------

project      = pimmp
configpath   = $(HOME)/.$(project)
importpath   = ardnew.com/$(project)
gopathsrc    = $(GOPATH)/src
gopathbin    = $(GOPATH)/bin

# -- define version info with version control ----------------------------------

version      = 0.1
revision     = r$(shell svn info| \grep -oP '^Revision:\s*\K\d+')
buildtime    = $(shell date -u '+%FT%TZ')

# -- compiler flags (see: go tool compile -help) -------------------------------

gcflags      =
gcflags-dbg  = all='-N -l'

# -- linker flags (see: go tool link -help) ------------------------------------

ldflags-vers = -X "main.identity=$(project)" -X "main.version=$(version)" -X "main.revision=$(revision)" -X "main.buildtime=$(buildtime)"
ldflags      = '-w -s $(ldflags-vers)'
ldflags-dbg  = '$(ldflags-vers)'



# ==============================================================================
#
#  TARGET DEFINITION
#
# ==============================================================================
#

# -- janitorial / cleanup targets ----------------------------------------------

.PHONY: clean clean-data clean-all

clean:
	rm -f "$(gopathsrc)/$(importpath)/$(project)"
	rm -f "$(gopathbin)/$(project)"
clean-data:
	rm -rf "$(configpath)"
clean-all: clean-data clean

# -- compilation targets -------------------------------------------------------

.PHONY: build build-dbg install install-dbg build-race build-dbg-race install-race install-dbg-race

build:
	go build -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"
build-dbg:
	go build -ldflags=$(ldflags-dbg) -gcflags=$(gcflags-dbg) '$(importpath)'
install:
	go install -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"
install-dbg:
	go install -ldflags=$(ldflags-dbg) -gcflags=$(gcflags-dbg) "$(importpath)"
build-race:
	go build -race -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"
build-dbg-race:
	go build -race -ldflags=$(ldflags-dbg) -gcflags=$(gcflags-dbg) '$(importpath)'
install-race:
	go install -race -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"
install-dbg-race:
	go install -race -ldflags=$(ldflags-dbg) -gcflags=$(gcflags-dbg) "$(importpath)"

# -- combined / composite targets ----------------------------------------------

.PHONY: clean-build clean-build-dbg clean-install clean-install-dbg clean-data-build clean-data-build-dbg clean-data-install clean-data-install-dbg clean-build-race clean-build-dbg-race clean-install-race clean-install-dbg-race clean-data-build-race clean-data-build-dbg-race clean-data-install-race clean-data-install-dbg-race

clean-build: clean build
clean-build-dbg: clean build-dbg
clean-install: clean install
clean-install-dbg: clean install-dbg
clean-data-build: clean-all build
clean-data-build-dbg: clean-all build-dbg
clean-data-install: clean-all install
clean-data-install-dbg: clean-all install-dbg
# including the race detector
clean-build-race: clean build-race
clean-build-dbg-race: clean build-dbg-race
clean-install-race: clean install-race
clean-install-dbg-race: clean install-dbg-race
clean-data-build-race: clean-all build-race
clean-data-build-dbg-race: clean-all build-dbg-race
clean-data-install-race: clean-all install-race
clean-data-install-dbg-race: clean-all install-dbg-race

# -- test / evaluation targets -------------------------------------------------

.PHONY: run

run:
	go run -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"

