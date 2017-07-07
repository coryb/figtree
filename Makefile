GENERATOR_SRC = \
	rawoption.go \
	$(NULL)

GENERATED_SRC = $(GENERATOR_SRC:%.go=gen-%.go)

test: $(GENERATED_SRC)
	go get -t -v
	go get github.com/kr/pretty
	go get gopkg.in/alecthomas/kingpin.v2
	go test

gen-%.go: %.go
	go get github.com/cheekybits/genny
	go generate

.PHONY: test
