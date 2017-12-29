all: upload-go serve

upload-go: src/upload-go.go
	GOPATH=$(PWD) go build $<

serve: src/serve.go
	GOPATH=$(PWD) go build $<

test:
	tox
