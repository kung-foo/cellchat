VERSION := v0.0.1
BUILDSTRING := $(shell git log --pretty=format:'%h' -n 1)
VERSIONSTRING := cellchat version $(VERSION)+$(BUILDSTRING)

ifndef GOARCH
	GOARCH := $(shell go env GOARCH)
endif

ifndef GOOS
	GOOS := $(shell go env GOOS)
endif

OUTPUT := cellchat-$(GOOS)-$(GOARCH)

ifeq ($(GOOS), windows)
	OUTPUT := $(OUTPUT).exe
endif

default: build

build: $(OUTPUT)

$(OUTPUT): main.go building.go room.go user.go topics.go server.go services.go
	godep go build -v -o $(OUTPUT) -ldflags "-X main.VERSION \"$(VERSIONSTRING)\"" .
	@echo Built ./$(OUTPUT)

gofmt:
	gofmt -w .

update-godeps:
	rm -rf Godeps
	godep save

test:
	godep go test -cover -v ./...

clean:
	rm -f $(OUTPUT)
