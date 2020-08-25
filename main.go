package main

import (
	"f.oxy.works/paulius.stundzia/tcpmessenger/messenger"
	"time"
)

func main() {
	msgr := messenger.GetMessenger(8033, 8044, 3)
	msgr.Run()
	for  {
		time.Sleep(2 * time.Second)
	}
}
