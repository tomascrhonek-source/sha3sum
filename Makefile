build:
	go vet
	go build
install:
	go install
run:
	go vet
	go build
	./sha3sum

