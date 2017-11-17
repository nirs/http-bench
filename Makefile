all: upload-go serve

upload-go: upload-go.go
	go build $<

serve: serve.go
	go build $<

test:
	tox
