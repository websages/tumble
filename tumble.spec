Name:	        tumble
Version:	==VERSION==
Release:	1%{?dist}
Summary:	A classic tumblelog application.

Group:		Internet/Applications
License:	ASL 2.0
URL:		http://tumble.wcyd.org
Source0:	%{name}-%{version}.tar.gz
BuildRoot:      %{_tmppath}/%{name}-%{version}-%{release}-%(id -un)

Requires:      perl(DBD::mysql)
Requires:      httpd mod_perl
Requires:      perl-CGI-Application
Requires:      perl-Crypt-SSLeay
# Only when running on localhost, but that's what's hard-coded for now.
Requires:      mysql-server
BuildArch:     noarch

%description
A classic tumblelog written in Perl in something like 2004.

%prep
%setup -q


%build


%install
make install DESTDIR=%{buildroot}

# Fix shebang line of scripts
for file in `find $RPM_BUILD_ROOT -type f`; do
echo $file
    sed -i -e '1s,^#!.*perl,#!%{_bindir}/perl,' $file
done


%files
%doc sql README.md
/srv/www/%{name}/htdocs/2202
/srv/www/%{name}/htdocs/*.png
/srv/www/%{name}/htdocs/buttons
/srv/www/%{name}/htdocs/css
/srv/www/%{name}/htdocs/favicon.ico
/srv/www/%{name}/htdocs/img
/srv/www/%{name}/htdocs/index.cgi
/srv/www/%{name}/htdocs/irclink
/srv/www/%{name}/htdocs/lib
/srv/www/%{name}/htdocs/quote
/srv/www/%{name}/htdocs/search.cgi
/srv/www/%{name}/htdocs/thtml
#%config(noreplace)%{_sysconfdir}/cron.hourly/*
%config(noreplace) /srv/www/%{name}/htdocs/config.yaml
%config(noreplace)%{_sysconfdir}/httpd/conf.d/*


%changelog
* Sun Oct 27 2013 <stahnma@websages.com> - 1.0.0-1
- First packaging
