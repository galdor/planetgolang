
gopkg= github.com/galdor/planetgolang
bin= planetgolang
build_id=$(shell git describe --always --dirty --long --tags)

installdir=
prefix= $(installdir)/usr/local
bindir= $(prefix)/bin
sharedir= $(prefix)/share/planetgolang
dbdir= $(installdir)/var/db/planetgolang

production=0
ifeq ($(production), 1)
	production_bool=true
else
	production_bool=false
endif

host= hades.snowsyn.net

all: build.go
	go build $(gopkg)

clean:
	$(RM) $(bin)
	$(RM) build.go

build:
	scp -q deployment/build $(host):/tmp/planetgolang-build
	ssh $(host) sh /tmp/planetgolang-build
	ssh $(host) rm /tmp/planetgolang-build
	scp -q '$(host):/tmp/planetgolang-*.txz' pkgs/

deploy:
	if [ -z "$(pkg)" ]; then echo "missing pkg"; exit 1; fi
	scp -q $(pkg) $(host):/tmp
	ssh root@$(host) pkg install -q -y /tmp/$(notdir $(pkg))

install:
	mkdir -p $(bindir)
	install -m 755 $(bin) $(bindir)
	mkdir -p $(sharedir)/www-data
	cp -r www-data/* $(sharedir)/www-data
	mkdir -p $(sharedir)/templates
	cp -r templates/* $(sharedir)/templates
	mkdir -p $(sharedir)/db
	cp -r db/* $(sharedir)/db
	mkdir -p $(dbdir)

uninstall:
	$(RM) $(bindir)/$(bin)
	$(RM) -r $(sharedir)

build.go:
	echo 'package main'                                  >$@
	echo 'const ('                                      >>$@
	echo '    BuildId string  = "$(build_id)"'          >>$@
	echo '    Production bool  = $(production_bool)'    >>$@
	echo '    DbDir string  = "$(dbdir)"'               >>$@
	echo '    ShareDir string  = "$(sharedir)"'         >>$@
	echo ')'                                            >>$@
	gofmt -w $@

.PHONY: all build build.go clean deploy install uninstall
