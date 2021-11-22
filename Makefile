GENERATOR_SRC = \
	rawoption.go \
	$(NULL)

GENERATED_SRC = $(GENERATOR_SRC:%.go=gen-%.go)

test: $(GENERATED_SRC)
	go test -v ./...

gen-%.go: %.go
	go generate ./...

.PHONY: test
