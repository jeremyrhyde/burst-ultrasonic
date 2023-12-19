
build: *.go 
	go build -o mergedcamera *.go

test:
	go test

lint:
	gofmt -w -s .

module.tgz: build 
	tar czf module.tgz mergedcamera
