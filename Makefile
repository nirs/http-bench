all: upload-go serve dd

.PHONY: upload-go serve dd

upload-go:
	GOPATH=$(PWD) go build src/upload-go.go

serve:
	GOPATH=$(PWD) go build src/serve.go

dd:
	GOPATH=$(PWD) go build src/dd.go

test:
	tox
