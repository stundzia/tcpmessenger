package messenger

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

var msgr *messenger

func init() {
	msgr = NewMessenger(8033)
	go msgr.Run()
}

func checkOutput(c net.Conn, expected []byte, t *testing.T) {
	out := make([]byte, 128)
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second)) // timeout in 3 seconds
	if _, err := c.Read(out); err == nil {
		out = bytes.Trim(out, "\x00")
		if bytes.Compare(out, expected) != 0 {
			t.Errorf("response did not match expected output - got `%s`, but expected `%s`", string(out), string(expected))
		}
	} else {
		t.Error("could not read from connection: ", err)
	}
}

func setupAndTestConnection(conn net.Conn, connType string, t *testing.T) {
	checkOutput(conn, []byte("Type `c` for `consumer`, `p` for producer\n"), t)
	if _, err := conn.Write([]byte(fmt.Sprintf("%s\n", connType))); err != nil {
		t.Error("could not write payload to server: ", err)
	}
	var connTypeFull string
	switch connType {
	case "p":
		connTypeFull = "producer"
	case "c":
		connTypeFull = "consumer"
	default:
		t.Error("bad test case, unknown connType provided: ", connType)
	}
	modeMsg := fmt.Sprintf("Entering `%s` mode\n", connTypeFull)
	checkOutput(conn, []byte(modeMsg), t)
}

func TestMessenger_Run(t *testing.T) {
	time.Sleep(1 * time.Second) // Give time for messenger to spin up.
	conn, err := net.Dial("tcp", ":8033")
	if err != nil || conn == nil {
		t.Error("could not obtain connection: ", err)
	}
	setupAndTestConnection(conn, "c", t)
}

func TestMessengerWithSingleProducerConsumerPair(t *testing.T) {
	tcs := []struct {
		test    string
		payload []byte
	}{
		{
			"Sending a msg to producer port sends the same message to consumer port",
			[]byte("Where's the money, Lebowski?\n"),
		},
		{
			"Sending a msg to producer port sends the same message to consumer port, case deux",
			[]byte("Where's the money, shithead?\n"),
		},
	}
	time.Sleep(1 * time.Second)
	for _, tc := range tcs {
		t.Run(tc.test, func(t *testing.T) {
			producerConn, err := net.Dial("tcp", ":8033")
			if err != nil || producerConn == nil {
				t.Error("could not obtain producer connection: ", err)
			}
			setupAndTestConnection(producerConn, "p", t)
			consumerConn, err := net.Dial("tcp", ":8033")
			if err != nil || consumerConn == nil {
				t.Error("could not obtain consumer connection: ", err)
			}
			setupAndTestConnection(consumerConn, "c", t)

			defer producerConn.Close()
			defer consumerConn.Close()

			if _, err = producerConn.Write(tc.payload); err != nil {
				t.Error("could not write payload to producer: ", err)
			}

			checkOutput(consumerConn, tc.payload, t)
		})
	}
}

func TestMessengerWithMultipleProducersAndConsumers(t *testing.T) {
	tcs := []struct {
		test     string
		payload1 []byte
		payload2 []byte
	}{
		{
			"Sending a msg via any producer connection, sends the same message to all consumer connections",
			[]byte("Where's the money, Lebowski?\n"),
			[]byte("Where's the money, shithead?\n"),
		},
		{
			"Sending a msg via any producer connection, sends the same message to all consumer connections, part deux",
			[]byte("Inconceivable!\n"),
			[]byte("You keep using that word. I do not think it means what you think it means.\n"),
		},
	}
	time.Sleep(1 * time.Second)
	for _, tc := range tcs {
		t.Run(tc.test, func(t *testing.T) {
			producerConn, err := net.Dial("tcp", ":8033")
			if err != nil || producerConn == nil {
				t.Error("could not obtain producer connection: ", err)
			}
			setupAndTestConnection(producerConn, "p", t)
			producerConn2, err := net.Dial("tcp", ":8033")
			if err != nil || producerConn2 == nil {
				t.Error("could not obtain producer connection no. 2: ", err)
			}
			setupAndTestConnection(producerConn2, "p", t)
			consumerConn, err := net.Dial("tcp", ":8033")
			if err != nil || consumerConn == nil {
				t.Error("could not obtain consumer connection: ", err)
			}
			setupAndTestConnection(consumerConn, "c", t)
			consumerConn2, err := net.Dial("tcp", ":8033")
			if err != nil || consumerConn2 == nil {
				t.Error("could not obtain consumer connection no. 2: ", err)
			}
			setupAndTestConnection(consumerConn2, "c", t)
			consumerConn3, err := net.Dial("tcp", ":8033")
			if err != nil || consumerConn3 == nil {
				t.Error("could not obtain consumer connection no. 3: ", err)
			}
			setupAndTestConnection(consumerConn3, "c", t)

			defer producerConn.Close()
			defer producerConn2.Close()
			defer consumerConn.Close()
			defer consumerConn2.Close()
			defer consumerConn3.Close()

			consumerConns := []net.Conn{consumerConn, consumerConn2, consumerConn3}

			if _, err := producerConn.Write(tc.payload1); err != nil {
				t.Error("could not write payload to producer: ", err)
			}

			for _, cc := range consumerConns {
				checkOutput(cc, tc.payload1, t)
			}

			if _, err := producerConn.Write(tc.payload2); err != nil {
				t.Error("could not write payload to producer: ", err)
			}

			for _, cc := range consumerConns {
				checkOutput(cc, tc.payload2, t)
			}
		})
	}
}
