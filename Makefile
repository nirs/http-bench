envs = $(wildcard py/*/bin/activate)

.PHONY: $(envs)

all: upload serve

upload: upload.go
	go build $<

serve: serve.go
	go build $<

test: $(envs)

$(envs):
	source $@; \
		pytest -vs test.py; \
		deactivate
