gdoc2pdf: *.go
	go build -o $@ $(buildflags)
clean:
	rm -f gdoc2pdf
.PHONY += clean
