.PHONY: build install release-dry-run clean

build:
	go build -ldflags "-X github.com/AngelMaldonado/cx/cmd.Version=dev" -o cx ./main.go

install: build
	cp cx /usr/local/bin/cx

release-dry-run:
	goreleaser release --snapshot --clean

clean:
	rm -rf dist/ cx
