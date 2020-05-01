buildflags := -ldflags "-w -s -X main.Version=$(shell git describe --abbrev=4 --dirty --always --tags)" -tags netgo,osusergo
gdoc2pdf: *.go
	go build -o $@ $(buildflags)
gdoc2pdf.exe: *.go
	env GOOS=windows GOARCH=amd64 go build $(buildflags) -o $@
gdoc2pdf-osx: *.go
	env GOOS=darwin GOARCH=amd64 go build $(buildflags) -o $@
checksums.txt:
	sha256sum gdoc2pdf* > checksums.txt
	gpg --armor --detach-sign checksums.txt
release: clean gdoc2pdf gdoc2pdf.exe gdoc2pdf-osx checksums.txt
clean:
	rm -f gdoc2pdf gdoc2pdf.exe gdoc2pdf-osx checksums.txt checksums.txt.asc
.PHONY += clean
