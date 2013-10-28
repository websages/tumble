
PKGNAME=tumble
TARDIR=/tmp
TMPDIR:=$(shell mktemp -d -u -p . -t rpmbuild-XXXXXXX)
DATADIR=$(DESTDIR)/srv/www/$(PKGNAME)
CONFDIR=$(DESTDIR)/etc/
APACHE_DIR=$(CONFDIR)/httpd/conf.d
SPEC_FILE=$(PKGNAME).spec

RPMBUILD := $(shell if test -f /usr/bin/rpmbuild ; then echo /usr/bin/rpmbuild ; else echo "x" ; fi)
RPM_DEFINES = --define "_specdir $(TMPDIR)/SPECS" --define "_rpmdir $(TMPDIR)/RPMS" --define "_sourcedir $(TMPDIR)/SOURCES" --define "_srcrpmdir $(TMPDIR)/SRPMS" --define "_builddir $(TMPDIR)/BUILD"
MAKE_DIRS= $(TMPDIR)/SPECS $(TMPDIR)/SOURCES $(TMPDIR)/BUILD $(TMPDIR)/SRPMS $(TMPDIR)/RPMS
VERSION=$(shell git describe | sed -e 's/-/\./g')
TARBALL=$(PKGNAME)-$(VERSION).tar.gz

install:
	#mkdir -p {$(DATADIR),$(CONFDIR),$(APACHE_DIR)}
	#install -p -m644 $(PKGNAME).conf $(APACHE_DIR)

tarball:
	mkdir -p $(TARDIR)
	cd ..; tar pczf $(TARDIR)/$(TARBALL) $(PKGNAME)
	mv $(TARDIR)/$(TARBALL) .

uninstall:
	rm -rf $(DATADIR)
	rm -rf $(APACHE_DIR)/$(PKGNAME).conf

clean:
	rm -f $(TARBALL)  *.rpm
	rm -rf BUILD SRPMS RPMS SPECS SOURCES
	rm -rf ./rpmbuild-*


srpm: tarball
	@mkdir -p $(MAKE_DIRS)
	cp -f $(TARBALL) $(TMPDIR)/SOURCES
	@wait
	$(RPMBUILD) $(RPM_DEFINES) -bs $(SPEC_FILE)
	@mv -f SRPMS/* .
	@rm -rf BUILD SRPMS RPMS SOURCES SPECS

tempdir:
	echo $(TMPDIR)
