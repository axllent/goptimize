TAG=`git describe --tags`
VERSION ?= `git describe --tags`
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}"

build = echo "\n\nBuilding $(1)-$(2)" && GOOS=$(1) GOARCH=$(2) go build ${LDFLAGS} -o dist/goptimize_${VERSION}_$(1)_$(2) \
	&& bzip2 dist/goptimize_${VERSION}_$(1)_$(2)

goptimize: *.go
	go get github.com/disintegration/imaging golang.org/x/image/bmp golang.org/x/image/tiff github.com/axllent/gitrel github.com/spf13/pflag
	go build ${LDFLAGS} -o goptimize
	rm -rf /tmp/go-*

clean:
	rm -f goptimize

release:
	mkdir -p dist
	rm -f dist/goptimize_${VERSION}_*
	go get github.com/disintegration/imaging golang.org/x/image/bmp golang.org/x/image/tiff github.com/axllent/gitrel github.com/spf13/pflag
	$(call build,linux,amd64)
	$(call build,linux,386)
	$(call build,linux,arm)
	$(call build,linux,arm64)
	$(call build,darwin,amd64)
	$(call build,darwin,386)
