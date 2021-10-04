Name:    dswatcher
Version: %{__version}
Release: %{__release}%{?dist}

License: GNU AGPLv3
URL: https://github.com/redBorder/dswatcher
Source0: %{name}-%{version}.tar.gz

BuildRequires: go glide
BuildRequires:	rsync
BuildRequires: librdkafka-devel

Requires: librdkafka1

Summary: Dynamic Sensors Watcher
Group:   Development/Libraries/Go

%description
%{summary}

%prep
%setup -qn %{name}-%{version}

%build
export GOPATH=${PWD}/gopath
export PATH=${GOPATH}:${PATH}
mkdir -p $GOPATH/src/github.com/redBorder/dswatcher
rsync -az --exclude=gopath/ ./ $GOPATH/src/github.com/redBorder/dswatcher
cd $GOPATH/src/github.com/redBorder/dswatcher
make

%install
export PARENT_BUILD=${PWD}
export GOPATH=${PWD}/gopath
export PATH=${GOPATH}:${PATH}
cd $GOPATH/src/github.com/redBorder/dswatcher
mkdir -p %{buildroot}/usr
prefix=%{buildroot}/usr make install
mkdir -p %{buildroot}/usr/share/dswatcher
mkdir -p %{buildroot}/etc/dswatcher
install -D -m 644 dswatcher.service %{buildroot}/usr/lib/systemd/system/dswatcher.service
install -D -m 644 packaging/rpm/config.yml %{buildroot}/usr/share/dswatcher

%clean
rm -rf %{buildroot}

%pre
getent group dswatcher >/dev/null || groupadd -r dswatcher
getent passwd dswatcher >/dev/null || \
    useradd -r -g dswatcher -d / -s /sbin/nologin \
    -c "User of dswatcher service" dswatcher
exit 0

%post -p /sbin/ldconfig
%postun -p /sbin/ldconfig

%files
%defattr(755,root,root)
/usr/bin/dswatcher
%defattr(644,root,root)
/usr/share/dswatcher/config.yml
/usr/lib/systemd/system/dswatcher.service

%changelog
* Mon Oct 04 2021 Miguel Negrón <manegron@redborder.com> & David Vanhoucke <dvanhoucke@redborder.com> - 1.0.0-1
- first spec version