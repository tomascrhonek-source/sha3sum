build:
	go vet
	go build
install:
	go install
run: build
	./sha3sum
package:
	go build
	cd build ; dpkg-deb --root-owner-group --build herons_sha3sum
