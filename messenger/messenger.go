package messenger

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

type messenger struct {
	msgPipeline            chan string
	producerPort           int
	consumerPort           int
	consumerConnectionPool []net.Conn
}

// GetMessenger initializes and returns a messenger instance.
func GetMessenger(producerPort int, consumerPort int) (msgr *messenger) {
	msgr = &messenger{
		msgPipeline:  make(chan string),
		producerPort: producerPort,
		consumerPort: consumerPort,
	}
}

// removeConnectionFromPool finds and removes the given connection from the messengers pool of consumer connections.
func (msgr *messenger) removeConnectionFromPool(c net.Conn) {
	for i, cc := range msgr.consumerConnectionPool {
		if c == cc {
			msgr.consumerConnectionPool = append(msgr.consumerConnectionPool[:i], msgr.consumerConnectionPool[i+1:]...)
		}
	}
}

// sendMessageToConsumerConnection sends the provided msg string over the given connection.
func (msgr *messenger) sendMessageToConsumerConnection(c net.Conn, msg string) {
	_, err := c.Write([]byte(msg))
	if err != nil {
		handleConnectionError(err, c)
		msgr.removeConnectionFromPool(c)
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

// listenForProducers listens for connections on the producer port.
func (msgr *messenger) listenForProducers() {
	port := ":" + strconv.Itoa(msgr.producerPort)
	l, err := net.Listen("tcp4", port)
	defer l.Close()
	if err != nil {
		log.Fatal(err)
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		go msgr.handleProducerConnection(c)
	}
}

// listenForConsumers listens for new consumer connections and adds them to the consumer connection pool.
func (msgr *messenger) listenForConsumers() {
	port := ":" + strconv.Itoa(msgr.consumerPort)
	l, err := net.Listen("tcp4", port)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		msgr.consumerConnectionPool = append(msgr.consumerConnectionPool, c)
	}
}

// produceMessages reads messages from the msgPipeline channel and sends them to active consumers.
func (msgr *messenger) produceMessages() {
	for {
		msg := <-msgr.msgPipeline + "\n"
		for _, c := range msgr.consumerConnectionPool {
			go msgr.sendMessageToConsumerConnection(c, msg)
		}
	}
}

// Run starts the messenger.
// Run will start listening for tcp connections on 2 ports (producerPort and consumerPort).
func (msgr *messenger) Run() {
	go msgr.listenForConsumers()
	go msgr.listenForProducers()
	go msgr.produceMessages()
}
