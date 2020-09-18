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
	msgPipeline            chan message
	port           int
	consumerConnectionPool map[net.Conn]string
	connectionPoolLock *sync.Mutex
	logger *zap.Logger
	chatNames map[string]struct{}
	chatNameLock *sync.Mutex
}

type message struct {
	name string
	content string
}

// outputString returns message string formatted for output.
func (msg message) outputString() string {
	if len(msg.name) > 0 {
		return fmt.Sprintf("%s: %s", msg.name, msg.content + "\n")
	}
	return msg.content + "\n"
}


// NewMessenger initializes and returns a messenger instance.
func NewMessenger(port int) (msgr *messenger) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	msgr = &messenger{
		msgPipeline:  make(chan message),
		port: port,
		consumerConnectionPool: make(map[net.Conn]string),
		connectionPoolLock: &sync.Mutex{},
		logger: logger,
		chatNames: make(map[string]struct{}),
		chatNameLock: &sync.Mutex{},
	}
	msgr.logger.Info("created new messenger", zap.Int("port", msgr.port))
	return
}


func (msgr *messenger) addConnectionAndNameToConsumers(c net.Conn, name string) error {
	if len(name) > 0 {
		msgr.chatNameLock.Lock()
		defer msgr.chatNameLock.Unlock()
		if _, exists := msgr.chatNames[name]; exists == true {
			return NewNameTakenError(name)
		}
		msgr.chatNames[name] = struct{}{}
	}
	msgr.connectionPoolLock.Lock()
	msgr.consumerConnectionPool[c] = name
	msgr.connectionPoolLock.Unlock()
	return nil
}

// removeConnectionFromPool finds and removes the given connection from the messengers pool of connections.
func (msgr *messenger) removeConnectionFromPool(pool map[net.Conn]string,c net.Conn) {
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
func (msgr *messenger) handleProducerConnection(c net.Conn, name string) {
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
		zap.String("name", name),
		)
		msgObj := message{
			name:    name,
			content: msg,
		}
		msgr.msgPipeline <- msgObj
		if err != nil {
			msgr.handleConnectionError(err, c)
			return
		}
	}
}

func (msgr *messenger) sortConnection(c net.Conn) {
	reader := bufio.NewReader(c)
	for {
		_, _ = c.Write([]byte("Type `c` for `consumer`, `p` for producer or `chat` for chat mode\n"))
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
				go msgr.handleProducerConnection(c, "")
				return
			case "c":
				_, _ = c.Write([]byte("Entering `consumer` mode\n"))
				msgr.logger.Info("registered new consumer", zap.String("address", c.RemoteAddr().String()))
				_ = msgr.addConnectionAndNameToConsumers(c, "")
				return
			case "chat":
				_, _ = c.Write([]byte("Entering `chat` mode, enter your name:\n"))
				name, err := reader.ReadString('\n')
				name = strings.Replace(name, "\n", "", -1)
				if err != nil {
					msgr.handleConnectionError(err, c)
					return
				}
				err = msgr.addConnectionAndNameToConsumers(c, name)
				if err != nil {
					_, _ = c.Write([]byte(err.Error() + "\n"))
					continue
				}

				go msgr.handleProducerConnection(c, name)
				msgr.logger.Info("registered new chat member", zap.String("address", c.RemoteAddr().String()), zap.String("name", name))
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
		msg := <-msgr.msgPipeline
		msgr.connectionPoolLock.Lock()
		for c, name := range msgr.consumerConnectionPool {
			if len(msg.name) > 0 && msg.name == name {
				continue
			}
			go msgr.sendMessageToConsumerConnection(c, msg.outputString())
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
