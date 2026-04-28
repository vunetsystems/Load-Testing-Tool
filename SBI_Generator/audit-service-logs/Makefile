.PHONY: build run clean

build:
	go build -o audit-generator cmd/generator/main.go

run: build
	./audit-generator --config config.yaml

clean:
	rm -f audit-generator
