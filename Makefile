
mergedcamera: *.go 
	go build -o mergedcamera *.go

test:
	go test

lint:
	gofmt -w -s .

module.tgz: mergedcamera 
	tar czf module.tgz mergedcamera
