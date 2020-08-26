package messenger

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

type Messenger struct {
	MsgPipeline            chan string
	producerPort           int
	consumerPort           int
	consumerConnectionPool []net.Conn
}

func GetMessenger(producerPort int, consumerPort int, msgBufferSize int) (msgr *Messenger) {
	msgr = &Messenger{
		MsgPipeline: make(chan string, msgBufferSize),
		producerPort: producerPort,
		consumerPort: consumerPort,
	}
	return
}

func (msgr *Messenger) removeConnectionFromPool(c net.Conn) {
	for i, cc := range msgr.consumerConnectionPool {
		if c == cc {
			msgr.consumerConnectionPool = append(msgr.consumerConnectionPool[:i], msgr.consumerConnectionPool[i+1:]...)
		}
	}
}

func (msgr *Messenger) sendMessageToConsumerConnection(c net.Conn, msg string) {
	_, err := c.Write([]byte(msg))
	if err != nil {
		log.Println(err)
		_ = c.Close()
		msgr.removeConnectionFromPool(c)
	}
}

func handleConnectionError(err error, c net.Conn) {
	log.Println(err)
	_ = c.Close()
}

func (msgr *Messenger) handleProducerConnection(c net.Conn) {
	defer c.Close()
	reader := bufio.NewReader(c)
	for {
		netData, err := reader.ReadString('\n')
		if err != nil {
			handleConnectionError(err, c)
			return
		}

		msg := strings.TrimSpace(netData)
		msgr.MsgPipeline <- msg
		_, err = c.Write([]byte("Acknowledged\n"))
		if err != nil {
			handleConnectionError(err, c)
			return
		}
	}
}

func (msgr *Messenger) listenForProducers() {
	port := ":" + strconv.Itoa(msgr.producerPort)
	l, err := net.Listen("tcp4", port)
	defer l.Close()
	if err != nil {
		fmt.Println(err)
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

// Listen for new consumer connections and append them to consumer connection pool.
func (msgr *Messenger) listenForConsumers() {
	port := ":" + strconv.Itoa(msgr.consumerPort)
	l, err := net.Listen("tcp4", port)
	if err != nil {
		fmt.Println(err)
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

func (msgr *Messenger) produceMessages() {
	for {
		msg := <- msgr.MsgPipeline + "\n"
		for _, c := range msgr.consumerConnectionPool {
			go msgr.sendMessageToConsumerConnection(c, msg)
		}
	}
}

func (msgr *Messenger) Run() {
	go msgr.listenForConsumers()
	go msgr.listenForProducers()
	go msgr.produceMessages()
}