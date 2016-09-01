
gopkg= github.com/galdor/planetgolang
bin= planetgolang
build_id=$(shell git describe --always --dirty --long --tags)

installdir=
prefix= $(installdir)/usr/local
bindir= $(prefix)/bin
sharedir= $(prefix)/share/planetgolang
dbdir= $(installdir)/var/db/planetgolang

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
	mkdir -p $(sharedir)/db
	cp -r db/* $(sharedir)/db
	mkdir -p $(dbdir)

uninstall:
	$(RM) $(bindir)/$(bin)
	$(RM) -r $(sharedir)

FORCE:
build.go: GNUmakefile
	echo 'package main'                                  >$@
	echo 'const ('                                      >>$@
	echo '    BuildId string  = "$(build_id)"'          >>$@
	echo '    DbDir string  = "$(dbdir)"'               >>$@
	echo ')'                                            >>$@
	gofmt -w $@

.PHONY: all build clean deploy install uninstall
