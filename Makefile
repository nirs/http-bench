all: upload serve

upload: upload.go
	go build $<

serve: serve.go
	go build $<

test:
	tox
