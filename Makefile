PKGNAME := foreman_ygg_worker
VERSION := 0.0.1
LIBEXECDIR := /usr/libexec
WORKER_USER := yggdrasil-worker

.PHONY: build
build: src/main.go src/server.go src/runner.go src/exec.go src/worker.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/foreman_worker src/main.go src/server.go src/runner.go src/exec.go src/worker.go

clean:
	rm -rf build

distribution-tarball:
	go mod vendor
	tar --create \
		--gzip \
		--file /tmp/$(PKGNAME)-$(VERSION).tar.gz \
		--exclude=.git \
		--exclude=.vscode \
		--exclude=.github \
		--exclude=.gitignore \
		--exclude=.copr \
		--transform s/^\./$(PKGNAME)-$(VERSION)/ \
		. && mv /tmp/$(PKGNAME)-$(VERSION).tar.gz .
	rm -rf ./vendor

.PHONY: data
data: build/com.redhat.Yggdrasil1.Worker1.foreman.conf build/com.redhat.Yggdrasil1.Worker1.foreman.service

build/%: %.in
	mkdir -p build
	sed \
		-e 's,[@]libexecdir[@],$(LIBEXECDIR),g' \
		-e 's,[@]worker_user[@],$(WORKER_USER),g' \
		$< > $@.tmp && mv $@.tmp $@

.PHONY: install
install: build data
	install -D -m 755 build/foreman_worker $(DESTDIR)/$(LIBEXECDIR)/foreman_worker
	install -D -m 644 build/com.redhat.Yggdrasil1.Worker1.foreman.conf $(DESTDIR)/usr/share/dbus-1/system.d/com.redhat.Yggdrasil1.Worker1.foreman.conf
	install -D -m 644 dbus_com.redhat.Yggdrasil1.Worker1.foreman.service $(DESTDIR)/usr/share/dbus-1/system-services/com.redhat.Yggdrasil1.Worker1.foreman.service
	install -D -m 644 build/com.redhat.Yggdrasil1.Worker1.foreman.service $(DESTDIR)/usr/lib/systemd/system/com.redhat.Yggdrasil1.Worker1.foreman.service
