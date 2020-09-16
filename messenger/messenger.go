package messenger

import (
	"bufio"
	"fmt"
	"go.uber.org/zap"
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
	logger *zap.Logger
}


// NewMessenger initializes and returns a messenger instance.
func NewMessenger(port int) (msgr *messenger) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	msgr = &messenger{
		msgPipeline:  make(chan string),
		port: port,
		consumerConnectionPool: make(map[net.Conn]struct{}),
		connectionPoolLock: &sync.Mutex{},
		logger: logger,
	}
	msgr.logger.Info("created new messenger", zap.Int("port", msgr.port))
	return
}

// removeConnectionFromPool finds and removes the given connection from the messengers pool of connections.
func (msgr *messenger) removeConnectionFromPool(pool map[net.Conn]struct{},c net.Conn) {
	msgr.connectionPoolLock.Lock()
	defer msgr.connectionPoolLock.Unlock()

	delete(pool, c)
	msgr.logger.Info(
		"removed connection from pool",
		zap.String("remote address", c.RemoteAddr().String()),
		)
}

// sendMessageToConsumerConnection sends the provided msg string over the given connection.
func (msgr *messenger) sendMessageToConsumerConnection(c net.Conn, msg string) {
	_, err := c.Write([]byte(msg))
	if err != nil {
		msgr.handleConnectionError(err, c)
		go msgr.removeConnectionFromPool(msgr.consumerConnectionPool, c)
	}
}

// handleConnectionError prints the error to stdout and closes the associated connection.
func (msgr *messenger) handleConnectionError(err error, c net.Conn) {
	msgr.logger.Error(
		fmt.Sprintf("connection error: %s", err.Error()),
		zap.String("remote address", c.RemoteAddr().String()),
		)
	_ = c.Close()
}

// handleProducerConnection reads messages sent via producer port connection and sends them to the messages channel.
func (msgr *messenger) handleProducerConnection(c net.Conn) {
	defer c.Close()
	reader := bufio.NewReader(c)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			msgr.handleConnectionError(err, c)
			return
		}

		msg = strings.TrimSpace(msg)
		msgr.logger.Debug(
			"message received",
			zap.String("producer", c.RemoteAddr().String()),
			zap.String("message content", msg),
			)
		msgr.msgPipeline <- msg
		_, err = c.Write([]byte("Acknowledged\n"))
		if err != nil {
			msgr.handleConnectionError(err, c)
			return
		}
	}
}

func (msgr *messenger) sortConnection(c net.Conn) {
	_, _ = c.Write([]byte("Type `c` for `consumer`, `p` for producer\n"))
	reader := bufio.NewReader(c)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			msgr.handleConnectionError(err, c)
			return
		}

		msg = strings.TrimSpace(msg)
		switch msg {
			case "p":
				_, _ = c.Write([]byte("Entering `producer` mode\n"))
				msgr.logger.Info("registered new producer", zap.String("address", c.RemoteAddr().String()))
				go msgr.handleProducerConnection(c)
				return
			case "c":
				_, _ = c.Write([]byte("Entering `consumer` mode\n"))
				msgr.logger.Info("registered new consumer", zap.String("address", c.RemoteAddr().String()))
				msgr.connectionPoolLock.Lock()
				msgr.consumerConnectionPool[c] = struct{}{}
				msgr.connectionPoolLock.Unlock()
				return
			default:
				_, err = c.Write([]byte("Unclear consumer/producer choice\n"))
		}
		if err != nil {
			msgr.handleConnectionError(err, c)
		}
	}
}


func (msgr *messenger) listenForConnections() {
	port := ":" + strconv.Itoa(msgr.port)
	l, err := net.Listen("tcp4", port)
	defer l.Close()
	if err != nil {
		msgr.logger.Fatal(fmt.Sprintf("unable to listen on provided port: %s", err.Error()), zap.Int("port", msgr.port))
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			msgr.logger.Error(fmt.Sprintf("unable to accept connection: %s", err.Error()))
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
	msgr.logger.Info("messenger running")
	go msgr.listenForConnections()
	go msgr.produceMessages()
}
