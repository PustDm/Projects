# Build for linux/riscv64 (Orange Pi RV2)
.PHONY: build-riscv64 build install

build:
	go build -o pico-dlna .

build-riscv64:
	GOOS=linux GOARCH=riscv64 go build -o pico-dlna-riscv64 .

install: build
	install -m 755 pico-dlna /usr/local/bin/
