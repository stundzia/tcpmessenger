package main

import (
	"f.oxy.works/paulius.stundzia/tcpmessenger/messenger"
)

func main() {
	msgr := messenger.GetMessenger(8033, 8044, 0)
	msgr.Run()
	select {}
}
