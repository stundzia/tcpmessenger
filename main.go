package main

import (
	"f.oxy.works/paulius.stundzia/tcpmessenger/messenger"
	"time"
)

func main() {
	msgr := messenger.GetMessenger(8033, 8044, 0)
	msgr.Run()
	for  {
		time.Sleep(2 * time.Second)
	}
}
