package main

import (
	"f.oxy.works/paulius.stundzia/tcpmessenger/messenger"
	"flag"
)

func main() {
	port := flag.Int("p", 8033, "Port (default: 8033)")
	flag.Parse()
	msgr := messenger.NewMessenger(*port)
	msgr.Run()
	select {}
}
