build:
	go vet
	go build
install:
	go install
run: build
	./sha3sum

