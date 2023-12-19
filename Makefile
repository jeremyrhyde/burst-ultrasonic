
build: *.go 
	go build -o burstultrasonic *.go

test:
	go test

lint:
	gofmt -w -s .

module.tgz: build 
	tar czf module.tgz burstultrasonic
