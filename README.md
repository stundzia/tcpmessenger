TCP Messenger
===

This is a simple TCP Messenger that listens on 2 ports, a "producer port" and
a "consumer port". Messages sent over a producer port connection will be forwarded
to any consumer port connections.

Tested with Go v1.14.

Download the package:
```
go get f.oxy.works/paulius.stundzia/tcpmessenger
```

Package usage:
```go
package main

import (
	"f.oxy.works/paulius.stundzia/tcpmessenger/messenger"
	"time"
)

func main() {
    // Create messenger that listens for messages on port 8033
    // and sends them to port 8044
	msgr := messenger.GetMessenger(8033, 8044)
    // Run the messenger
	msgr.Run()
    // Prevent main goroutine from exiting
	select {}
}
```

Build and run in docker:
```
# Build from within tcpmessenger root directory:
docker build --tag tcpmessenger .
# Run container:
docker run -p 8033:8033 -p 8044:8044 tcpmessenger
```