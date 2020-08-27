package main

import (
	"f.oxy.works/paulius.stundzia/tcpmessenger/messenger"
	"flag"
)

func main() {
	producerPort := flag.Int("p", 8033, "Producer port (default: 8033)")
	consumerPort := flag.Int("c", 8044, "Consumer port (default: 8044)")
	flag.Parse()
	msgr := messenger.GetMessenger(*producerPort, *consumerPort)
	msgr.Run()
	select {}
}
