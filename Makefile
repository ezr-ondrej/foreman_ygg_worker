PKGNAME := foreman_ygg_worker
VERSION := 0.0.1

build: src/main.go src/server.go src/runner.go
	mkdir -p build
	CGO_ENABLED=0 go build -o build/foreman_worker src/main.go src/server.go src/runner.go

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
