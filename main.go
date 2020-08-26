package main

import (
	"f.oxy.works/paulius.stundzia/tcpmessenger/messenger"
)

func main() {
	msgr := messenger.GetMessenger(8033, 8044)
	msgr.Run()
	select {}
}
