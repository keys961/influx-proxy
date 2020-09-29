### Makefile --- 

## Author: Shell.Xu
## Version: $Id: Makefile,v 0.0 2017/01/17 03:44:24 shell Exp $
## Copyright: 2017, Eleme <zhixiang.xu@ele.me>
## License: MIT
## Keywords: 
## X-URL: 

all: build

build:
	mkdir -p bin
	go build -o bin/influx-proxy .

test:
	go test -v ./backend
	rm -rf ./backend/*.dat
	rm -rf ./backend/*.rec
	rm -rf ./*.dat
	rm -rf ./*.rec


bench:
	go test -bench=. ./backend

clean:
	rm -rf bin


### Makefile ends here
