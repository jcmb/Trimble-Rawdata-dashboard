module github.com/gkirk/trimble-rawdata-dashboard

go 1.22

require bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol v0.0.0

require (
	github.com/creack/goselect v0.1.2 // indirect
	go.bug.st/serial v1.6.2 // indirect
	golang.org/x/sys v0.19.0 // indirect
)

replace bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol => ../../BitBucket/geoffrey-kirk-go-dcol
