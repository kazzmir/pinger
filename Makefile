pinger: pinger.go
	go build .

.PHONY: setup

setup:
	go get github.com/sparrc/go-ping
	go get github.com/nsf/termbox-go
