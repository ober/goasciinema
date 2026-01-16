BINARY_NAME := goasciinema
PREFIX := ~/.local

.PHONY: build install clean

build:
	go build -o $(BINARY_NAME) .

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BINARY_NAME) $(DESTDIR)$(PREFIX)/bin/

clean:
	rm -f $(BINARY_NAME)
