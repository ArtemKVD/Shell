.PHONY: build run deb clean

build:
	go build -o kubsh cmd/kubsh/main.go

run: build
	./kubsh

deb: build
	dpkg-buildpackage -b -uc -us -d

clean:
	rm -f kubsh
	rm -rf debian/kubsh
	rm -f ../kubsh_*.deb
	rm -f ../kubsh_*.changes
	rm -f ../kubsh_*.buildinfo