# static project definitions
project      = pimmp
importpath   = ardnew.com/$(project)
gopathsrc    = $(GOPATH)/src
gopathbin    = $(GOPATH)/bin

# grab version info from revision control
version      = 0.1
revision     = r$(shell svn info| \grep -oP '^Revision:\s*\K\d+')
buildtime    = $(shell date -u '+%FT%TZ')

# compiler flags (see: go tool compile -help)
gcflags      =
gcflags-dbg  = all='-N -l'

# linker flags (see: go tool link -help)
ldflags-vers = -X "main.version=$(version)" -X "main.revision=$(revision)" -X "main.buildtime=$(buildtime)"
ldflags      = '-w -s $(ldflags-vers)'
ldflags-dbg  = '$(ldflags-vers)'

.PHONY: build
build:
	go build -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"

.PHONY: build-dbg
build-dbg:
	go build -ldflags=$(ldflags-dbg) -gcflags=$(gcflags-dbg) '$(importpath)'

.PHONY: install
install:
	go install -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"

.PHONY: install-dbg
install-dbg:
	go install -ldflags=$(ldflags-dbg) -gcflags=$(gcflags-dbg) "$(importpath)"

.PHONY: run
run:
	go run -ldflags=$(ldflags) -gcflags=$(gcflags) "$(importpath)"

.PHONY: clean
clean:
	rm -f "$(gopathsrc)/$(importpath)/$(project)"
	rm -f "$(gopathbin)/$(project)"


