
PKGNAME=tumble
TMP_PATTERN:=$(shell mktemp -d -u -p . -t rpmbuild-XXXXXXX)
TMPDIR=$(shell pwd)/$(TMP_PATTERN)
TAR_TMP_DIR:=$(shell mktemp -d -u -t tarball-XXXXXXX)

DATADIR=$(DESTDIR)/srv/www/$(PKGNAME)
CONFDIR=$(DESTDIR)/etc/
CRON_DIR=$(CONFDIR)/cron.hourly
SPEC_FILE=$(PKGNAME).spec

RPMBUILD := $(shell if test -f /usr/bin/rpmbuild ; then echo /usr/bin/rpmbuild ; else echo "x" ; fi)
RPM_DEFINES = --define "_specdir $(TMPDIR)/SPECS" --define "_rpmdir $(TMPDIR)/RPMS" --define "_sourcedir $(TMPDIR)/SOURCES" --define "_srcrpmdir $(TMPDIR)/SRPMS" --define "_builddir $(TMPDIR)/BUILD"
MAKE_DIRS= $(TMPDIR)/SPECS $(TMPDIR)/SOURCES $(TMPDIR)/BUILD $(TMPDIR)/SRPMS $(TMPDIR)/RPMS
VERSION=$(shell git describe | sed -e 's/-/\./g')
TARBALL=$(PKGNAME)-$(VERSION).tar.gz

DEBIAN :=$(shell test -f "/etc/debian_version" && echo 'debian' || echo 'x')

ifeq ($(DEBIAN), debian)
APACHE_DIR=$(CONFDIR)apache2/sites-available/
else
APACHE_DIR=$(CONFDIR)httpd/conf.d
endif


build:
	#go build -o scripts/twit-link scripts/twit-link.go


install:
	mkdir -p $(DATADIR) $(APACHE_DIR) $(CONFDIR)/$(PKGNAME)
	install -p -m644 htdocs/config.yaml $(CONFDIR)/$(PKGNAME)
	cp -pr htdocs $(DATADIR)
	mkdir -p $(DESTDIR)/usr/local/bin
	#go build -o scripts/twit-link scripts/twit-link.go
	#cp -pr scripts/twit-link $(DESTDIR)/usr/local/bin

tarball: clean
	mkdir -p $(TAR_TMP_DIR)/$(PKGNAME)-$(VERSION)
	cd ..; cp -pr $(PKGNAME)/* $(TAR_TMP_DIR)/$(PKGNAME)-$(VERSION)
	cd $(TAR_TMP_DIR); tar pczf $(TARBALL)  $(PKGNAME)-$(VERSION)
	mv $(TAR_TMP_DIR)/$(TARBALL) .
	rm -rf $(TAR_TMP_DIR)

uninstall:
	rm -rf $(DATADIR)
	rm -rf $(APACHE_DIR)/$(PKGNAME).conf

clean:
	rm -f $(TARBALL)  *.rpm
	rm -f scripts/twit-link twit-link
	rm -rf debian/changelog debian/$(PKGNAME)* debian/tmp debian/files
	rm -rf BUILD SRPMS RPMS SPECS SOURCES
	rm -rf ./rpmbuild-* ./tarball-* ./$(PKGNAME)*gz


srpm: tarball
	@mkdir -p $(MAKE_DIRS)
	cp -f $(TARBALL) $(TMPDIR)/SOURCES
	cp -f $(SPEC_FILE) $(TMPDIR)/SPECS
	sed -i 's/==VERSION==/$(VERSION)/g' $(TMPDIR)/SPECS/$(SPEC_FILE)
	@wait
	$(RPMBUILD) $(RPM_DEFINES) -bs $(TMPDIR)/SPECS/$(SPEC_FILE)
	@mv -f $(TMPDIR)/SRPMS/* .
	@rm -rf $(TMPDIR)

deb:
	sed -e 's/==VERSION==/$(VERSION)/g' debian/changelog.in > debian/changelog
	@wait
	dpkg-buildpackage

rpm: clean tarball
	@mkdir -p $(MAKE_DIRS)
	cp -f $(TARBALL) $(TMPDIR)/SOURCES
	cp -f $(SPEC_FILE) $(TMPDIR)/SPECS
	sed -i 's/==VERSION==/$(VERSION)/g' $(TMPDIR)/SPECS/$(SPEC_FILE)
	@wait
	$(RPMBUILD) $(RPM_DEFINES) -ba $(TMPDIR)/SPECS/$(SPEC_FILE)
	@mv -f $(TMPDIR)/RPMS/noarch/* .
	@rm -rf $(TMPDIR)

tempdir:
	echo $(TMPDIR)
