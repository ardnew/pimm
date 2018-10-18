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

.PHONY: clean clean-data

clean:
	rm -f "$(gopathsrc)/$(importpath)/$(project)"
	rm -f "$(gopathbin)/$(project)"
clean-data: clean
	rm -rf "$(configpath)"

# -- compilation targets -------------------------------------------------------

.PHONY: build build-dbg install install-dbg

build:
	go build -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"
build-dbg:
	go build -ldflags=$(ldflags-dbg) -gcflags=$(gcflags-dbg) '$(importpath)'
install:
	go install -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"
install-dbg:
	go install -ldflags=$(ldflags-dbg) -gcflags=$(gcflags-dbg) "$(importpath)"

# -- combined / composite targets ----------------------------------------------

.PHONY: clean-build clean-build-dbg clean-install clean-install-dbg clean-data-build clean-data-build-dbg clean-data-install clean-data-install-dbg

clean-build: clean build
clean-build-dbg: clean build-dbg
clean-install: clean install
clean-install-dbg: clean install-dbg
clean-data-build: clean-data build
clean-data-build-dbg: clean-data build-dbg
clean-data-install: clean-data install
clean-data-install-dbg: clean-data install-dbg

# -- test / evaluation targets -------------------------------------------------

.PHONY: run

run:
	go run -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"

