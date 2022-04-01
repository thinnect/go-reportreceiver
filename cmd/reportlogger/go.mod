module github.com/thinnect/go-reportreceiver/cmd/reportlogger

go 1.18

replace github.com/thinnect/go-reportreceiver => ../..

require (
	github.com/jessevdk/go-flags v1.5.0
	github.com/proactivity-lab/go-loggers v0.0.0-20180417085828-f892709079bd
	github.com/proactivity-lab/go-moteconnection v0.0.2
	github.com/thinnect/go-reportreceiver v0.0.0-00010101000000-000000000000
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/joaojeronimo/go-crc16 v0.0.0-20140729130949-59bd0194935e // indirect
	go.bug.st/serial.v1 v0.0.0-20191202182710-24a6610f0541 // indirect
	golang.org/x/sys v0.0.0-20210320140829-1e4c9ba3b0c4 // indirect
)
