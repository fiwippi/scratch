build:
	go build -o ./bin/halo cmd/*.go

clean:
	rm -rf ./data ./bin
	
run:
	mkdir -p ./data
	./bin/halo -config config.toml

test:
	go test -v -race -cover

lint:
	golangci-lint run ./...
