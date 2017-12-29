all: upload-go serve

.PHONY: upload-go serve

upload-go:
	GOPATH=$(PWD) go build src/upload-go.go

serve:
	GOPATH=$(PWD) go build src/serve.go

test:
	tox
