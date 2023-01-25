MK_OS?=$(shell go env GOOS)
MK_ARCH?=$(shell go env GOARCH)
VERSION=$(shell git log -1 --date=format:"%Y%m%d.%H%M" --pretty='format:%h.%ad')
BUILDDIR=out
BIN=./$(BUILDDIR)/$(MK_OS)-$(MK_ARCH)/slides

run: build
	$(BIN) -static-path ./cmd/slides/pkg/static/ run

build: $(BIN)

linux:
	MK_OS=linux MK_ARCH=amd64 make build

$(BUILDDIR):
	mkdir -p $@

$(BUILDDIR)/%/slides: $(BUILDDIR) pkg/* cmd/*
	go vet ./...
	go test ./...
	GOOS=$(MK_OS) GOARCH=$(MK_ARCH) go build \
	  -ldflags "-X main.version=$(VERSION)" \
	  -o $@ ./cmd/slides

clean:
	rm -rf $(BUILDDIR)
