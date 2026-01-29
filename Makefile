.PHONY: build-ui build run test clean

build-ui:
	cd react-client && npm install && npm run build

build:
	go build -o bin/sqliter ./cmd/sqliter

run: build
	./bin/sqliter sample_data

test:
	go test ./...

clean:
	rm -rf bin/
	rm -rf server/ui/*
