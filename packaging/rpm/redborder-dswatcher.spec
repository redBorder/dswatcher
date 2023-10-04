Name:    redborder-dswatcher
Version: %{__version}
Release: %{__release}%{?dist}

License: GNU AGPLv3
URL: https://github.com/redBorder/dswatcher
Source0: %{name}-%{version}.tar.gz

BuildRequires: go gcc git rsync pkgconfig librd-devel librdkafka-devel
Requires: librd0 librdkafka

Summary: Dynamic Sensors Watcher
Group:   Development/Libraries/Go

%description
%{summary}

%prep
%setup -qn %{name}-%{version}

%build
git config --global --add safe.directory /builddir/build/BUILD
export PKG_CONFIG_PATH=/usr/lib/pkgconfig
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
export PKG_CONFIG_PATH=/usr/lib64/pkgconfig

cd $GOPATH/src/github.com/redBorder/dswatcher
mkdir -p %{buildroot}/usr/bin
prefix=%{buildroot}/usr PKG_CONFIG_PATH=/usr/lib/pkgconfig/ make install
mkdir -p %{buildroot}/usr/share/redborder-dswatcher
mkdir -p %{buildroot}/etc/redborder-dswatcher
install -D -m 644 redborder-dswatcher.service %{buildroot}/usr/lib/systemd/system/redborder-dswatcher.service
install -D -m 644 packaging/rpm/config.yml %{buildroot}/usr/share/redborder-dswatcher

%clean
rm -rf %{buildroot}

%pre
getent group redborder-dswatcher >/dev/null || groupadd -r redborder-dswatcher
getent passwd redborder-dswatcher >/dev/null || \
    useradd -r -g redborder-dswatcher -d / -s /sbin/nologin \
    -c "User of redborder-dswatcher service" redborder-dswatcher
exit 0

%post -p /sbin/ldconfig
%postun -p /sbin/ldconfig
systemctl daemon-reload

%files
%defattr(755,root,root)
/usr/bin/redborder-dswatcher
%defattr(644,root,root)
/usr/share/redborder-dswatcher/config.yml
/usr/lib/systemd/system/redborder-dswatcher.service

%changelog
* Wed Oct 04 2023 David Vanhoucke <dvanhoucke@redborder.com> - 2.0.0-1
- adapt for go mod
* Mon Oct 04 2021 Miguel Negrón <manegron@redborder.com> & David Vanhoucke <dvanhoucke@redborder.com> - 1.0.0-1
- first spec version