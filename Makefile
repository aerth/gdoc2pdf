buildflags := -ldflags "-w -s -X main.Version=$(shell git describe --abbrev=4 --dirty --always --tags)" -tags netgo,osusergo
gdoc2pdf: *.go
	go build -o $@ $(buildflags)
clean:
	rm -f gdoc2pdf
.PHONY += clean
