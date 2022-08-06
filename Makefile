BINARY="deliverbot"
VERSION=latest #v0.0.0
BUILD=`date +%FT%T%z`
PACKAGES=`go list ./... | grep -v /vendor/`
VETPACKAGES=`go list ./... | grep -v /vendor/ | grep -v /${BINARY}/`
GOFILES=`find . -name "*.go" -type f -not -path "./vendor/*"`
LDFlags="-X 'main.Version=${VERSION}' -X 'main.Commit=`git rev-parse HEAD`' -extldflags '-static'"
default:
	@go build -o ${BINARY} -a -ldflags ${LDFlags}
linux:
	@echo "build linux version"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${BINARY} -a -ldflags ${LDFlags} .
darwin:
	@echo "build darwin version"
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ${BINARY} -a -ldflags ${LDFlags} .
docker-build-linux:
	@echo "docker build linux version"
	@docker run -v ${GOPATH}:/go --rm -v ${PWD}:/go/src/${BINARY} -w /go/src/${BINARY} -e GOOS=linux -e GOARCH=amd64 golang:latest make linux
docker-build-darwin:
	@echo "docker build darwin version"
	@docker run -v ${GOPATH}:/go --rm -v ${PWD}:/go/src/${BINARY} -w /go/src/${BINARY} -e GOOS=darwin -e GOARCH=amd64 golang:latest make darwin
list:
	@echo ${PACKAGES}
fmt:
	@gofmt -s -w ${GOFILES}
fmt-check:
	@diff=$$(gofmt -s -d $(GOFILES)); \
	if [ -n "$$diff" ]; then \
		echo "Please run 'make fmt' and commit the result:"; \
		echo "$${diff}"; \
		exit 1; \
	fi;
install:
	@go install -a -ldflags ${LDFlags}
test:
	@go test -cpu=1,2,4 -v -tags integration ./...
vet:
	@go vet $(VETPACKAGES)
docker:
	@docker build -t nonedotone/${BINARY}:${VERSION} .
docker-push:
	@docker push nonedotone/${BINARY}:${VERSION}
clean:
	@if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
.PHONY: default linux darwin docker-build-linux docker-build-darwin list fmt fmt-check install test vet docker clean