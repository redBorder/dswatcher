Name:    dswatcher
Version: %{__version}
Release: %{__release}%{?dist}

License: GNU AGPLv3
URL: https://github.com/redBorder/dswatcher
Source0: %{name}-%{version}.tar.gz

BuildRequires: go = 1.6.3
BuildRequires: glide rsync gcc git
BuildRequires:	rsync
BuildRequires: librd-devel = 0.1.0
BuildRequires: librdkafka-devel = 0.9.1

Requires: librd0 librdkafka1

Summary: Dynamic Sensors Watcher
Group:   Development/Libraries/Go

%description
%{summary}

%prep
%setup -qn %{name}-%{version}

%build
git clone --branch v1.0.0-RC9 https://github.com/edenhill/librdkafka.git /tmp/librdkafka-v1.0.0-RC9
cd /tmp/librdkafka-v1.0.0-RC9
make uninstall
./configure && make
make install
cd -
ldconfig
export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig
ls /usr/local/lib/pkgconfig
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
export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig
cd $GOPATH/src/github.com/redBorder/dswatcher
mkdir -p %{buildroot}/usr/bin
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