build:
	go vet
	go build
install:
	go install
run: build
	./sha3sum
package:
	go build
	mv sha3sum build/sha3sum/usr/bin/
	cd build ; dpkg-deb --root-owner-group --build sha3sum
