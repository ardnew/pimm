project    = pimm
importpath = ardnew.com/$(project)

version    = 0.1
revision   = r$(shell svn info| \grep -oP '^Revision:\s*\K\d+')
buildtime  = $(shell date -u '+%FT%TZ')

ldflags    = -ldflags '-w -s -X "main.VERSION=$(version)" -X "main.REVISION=$(revision)" -X "main.BUILDTIME=$(buildtime)"'

.PHONY: build
build:
	go build $(ldflags) "$(importpath)"

.PHONY: install
install:
	go install $(ldflags) "$(importpath)"