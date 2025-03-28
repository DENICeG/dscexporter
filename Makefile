APPNAME := dscexporter
APPSRC := ./cmd/$(APPNAME)

PKG := github.com/DENICeG/dscexporter
TEST_PKG_LIST := config dscparser exporters scheduler

GITCOMMITHASH := $(shell git log --max-count=1 --pretty="format:%h" HEAD)
GITCOMMIT := -X main.gitcommit=$(GITCOMMITHASH)

VERSIONTAG := $(shell git describe --tags --abbrev=0)
VERSION := -X main.appversion=$(VERSIONTAG)

BUILDTIMEVALUE := $(shell date +%Y-%m-%dT%H:%M:%S%z)
BUILDTIME := -X main.buildtime=$(BUILDTIMEVALUE)

LDFLAGS := '-extldflags "-static" -d -s -w $(GITCOMMIT) $(VERSION) $(BUILDTIME)'
LDFLAGS_WINDOWS := '-extldflags "-static" -s -w $(GITCOMMIT) $(VERSION) $(BUILDTIME)'

clean:
	rm -rf build

info: 
	@echo - appname:   $(APPNAME)
	@echo - verison:   $(VERSIONTAG)
	@echo - commit:    $(GITCOMMITHASH)
	@echo - buildtime: $(BUILDTIMEVALUE) 

dep:
	@go get -v -d ./...

install: build-linux
	cp build/linux/$(APPNAME) $(shell go env GOPATH)/bin/
	
build-linux: info dep
	@echo Building for linux
	@mkdir -p build/linux
	@CGO_ENABLED=0 \
	GOOS=linux \
	go build -o build/linux/$(APPNAME)-$(VERSIONTAG)-$(GITCOMMITHASH) -a -ldflags $(LDFLAGS) $(APPSRC)
	@cp build/linux/$(APPNAME)-$(VERSIONTAG)-$(GITCOMMITHASH) build/linux/$(APPNAME)

unittests:
	CGO_ENABLED=0 go test -v -count=1 -cover -coverprofile cover.out -p 1 $(addprefix $(PKG)/, $(TEST_PKG_LIST))

test: unittests
