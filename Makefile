# Имя бинарника
BINARY := telnet
# Пакет с main.go (текущая папка)
PKG := .
# Параметры для запуска
HOST ?= tcpbin.com
PORT ?= 4242
TIMEOUT ?= 10s


.PHONY: all build run clean fmt vet lint help


all: build


build:
	go build -o $(BINARY) $(PKG)


run: build
	./$(BINARY) --timeout=$(TIMEOUT) $(HOST) $(PORT)


fmt:
	go fmt ./...


vet:
	go vet ./...


lint:
	@golangci-lint run || echo "(optional) install golangci-lint: https://golangci-lint.run/"


clean:
	rm -f $(BINARY)