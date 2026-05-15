.PHONY: build build-riscv install clean

GO ?= go
BINARY := pico-samba

build:
	$(GO) build -o $(BINARY) ./cmd/smb2

build-riscv:
	GOOS=linux GOARCH=riscv64 $(GO) build -o $(BINARY)-riscv64 ./cmd/smb2

install: build
	install -m 755 $(BINARY) /usr/local/bin/
	install -m 644 pico-samba.service /etc/systemd/system/
	systemctl daemon-reload

clean:
	rm -f $(BINARY) $(BINARY)-riscv64 *.exe
