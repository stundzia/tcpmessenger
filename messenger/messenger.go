package messenger

import (
	"bufio"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

type messenger struct {
	msgPipeline            chan string
	port           int
	consumerConnectionPool map[net.Conn]struct{}
	connectionPoolLock *sync.Mutex
}


// NewMessenger initializes and returns a messenger instance.
func NewMessenger(port int) (msgr *messenger) {
	msgr = &messenger{
		msgPipeline:  make(chan string),
		port: port,
		consumerConnectionPool: make(map[net.Conn]struct{}),
		connectionPoolLock: &sync.Mutex{},
	}
	return
}

// removeConnectionFromPool finds and removes the given connection from the messengers pool of connections.
func (msgr *messenger) removeConnectionFromPool(pool map[net.Conn]struct{},c net.Conn) {
	msgr.connectionPoolLock.Lock()
	defer msgr.connectionPoolLock.Unlock()

	delete(pool, c)
}

// sendMessageToConsumerConnection sends the provided msg string over the given connection.
func (msgr *messenger) sendMessageToConsumerConnection(c net.Conn, msg string) {
	_, err := c.Write([]byte(msg))
	if err != nil {
		handleConnectionError(err, c)
		go msgr.removeConnectionFromPool(msgr.consumerConnectionPool, c)
	}
}

// handleConnectionError prints the error to stdout and closes the associated connection.
func handleConnectionError(err error, c net.Conn) {
	log.Println(err)
	_ = c.Close()
}

// handleProducerConnection reads messages sent via producer port connection and sends them to the messages channel.
func (msgr *messenger) handleProducerConnection(c net.Conn) {
	defer c.Close()
	reader := bufio.NewReader(c)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			handleConnectionError(err, c)
			return
		}

		msg = strings.TrimSpace(msg)
		msgr.msgPipeline <- msg
		_, err = c.Write([]byte("Acknowledged\n"))
		if err != nil {
			handleConnectionError(err, c)
			return
		}
	}
}

func (msgr *messenger) sortConnection(c net.Conn) {
	reader := bufio.NewReader(c)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			handleConnectionError(err, c)
			return
		}

		msg = strings.TrimSpace(msg)
		switch msg {
			case "p":
				go msgr.handleProducerConnection(c)
				return
			case "c":
				msgr.consumerConnectionPool[c] = struct{}{}
				return
			default:
				_, err = c.Write([]byte("Unclear consumer/producer choice\n"))
		}
		if err != nil {
			handleConnectionError(err, c)
		}
	}
}


func (msgr *messenger) listenForConnections() {
	port := ":" + strconv.Itoa(msgr.port)
	l, err := net.Listen("tcp4", port)
	defer l.Close()
	if err != nil {
		log.Fatal(err)
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			return
		}
		go msgr.sortConnection(c)
	}
}

// produceMessages reads messages from the msgPipeline channel and sends them to active consumers.
func (msgr *messenger) produceMessages() {
	for {
		msg := <-msgr.msgPipeline + "\n"
		msgr.connectionPoolLock.Lock()
		for c, _ := range msgr.consumerConnectionPool {
			go msgr.sendMessageToConsumerConnection(c, msg)
		}
		msgr.connectionPoolLock.Unlock()
	}
}

// Run starts the messenger.
// Run will start listening for tcp connections on 2 ports (producerPort and consumerPort).
func (msgr *messenger) Run() {
	go msgr.listenForConnections()
	go msgr.produceMessages()
}
