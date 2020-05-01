buildflags := -ldflags "-X main.Version=$(shell git describe --abbrev=4 --dirty --always --tags)"
gdoc2pdf: *.go
	go build -o $@ $(buildflags)
clean:
	rm -f gdoc2pdf
.PHONY += clean
