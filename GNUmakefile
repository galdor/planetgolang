
pkg= github.com/galdor/planetgolang
bin= planetgolang
build_id=$(shell git describe --always --dirty --long --tags)

prefix= /usr/local
bindir= $(prefix)/bin
dbdir= /var/db/planetgolang
sharedir= $(prefix)/share/planetgolang

all: build.go
	go build $(pkg)

clean:
	$(RM) $(bin)
	$(RM) build.go

install:
	mkdir -p $(bindir)
	install -m 755 $(bin) $(bindir)
	mkdir -p $(sharedir)/www-data
	cp -r www-data/* $(sharedir)/www-data

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

.PHONY: all clean install uninstall
