package messenger

import (
	"bytes"
	"net"
	"testing"
	"time"
)

var msgr *messenger

func init() {
	msgr = GetMessenger(8033, 8044)
	go func() {
		msgr.Run()
	}()
}

func checkOutputOnConsumerConnection(c net.Conn, expected []byte, t *testing.T) {
	out := make([]byte, 128)
	if _, err := c.Read(out); err == nil {
		out = bytes.Trim(out, "\x00")
		if bytes.Compare(out, expected) != 0 {
			t.Errorf("response did not match expected output - got `%s`, but expected `%s`", string(out), string(expected))
		}
	} else {
		t.Error("could not read from connection")
	}
}

func TestMessenger_Run(t *testing.T) {
	time.Sleep(1 * time.Second) // Give time for messenger to spin up.
	producerConn, err := net.Dial("tcp", ":8033")
	if err != nil || producerConn == nil {
		t.Error("could not obtain producer connection: ", err)
	}
	consumerConn, err := net.Dial("tcp", ":8044")
	if err != nil || consumerConn == nil {
		t.Error("could not obtain consumer connection: ", err)
	}
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
			consumerConn, err := net.Dial("tcp", ":8044")
			if err != nil || consumerConn == nil {
				t.Error("could not obtain consumer connection: ", err)
			}

			defer producerConn.Close()
			defer consumerConn.Close()

			if _, err := producerConn.Write(tc.payload); err != nil {
				t.Error("could not write payload to producer: ", err)
			}

			checkOutputOnConsumerConnection(consumerConn, tc.payload, t)
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
			producerConn2, err := net.Dial("tcp", ":8033")
			if err != nil || producerConn2 == nil {
				t.Error("could not obtain producer connection no. 2: ", err)
			}
			consumerConn, err := net.Dial("tcp", ":8044")
			if err != nil || consumerConn == nil {
				t.Error("could not obtain consumer connection: ", err)
			}
			consumerConn2, err := net.Dial("tcp", ":8044")
			if err != nil || consumerConn2 == nil {
				t.Error("could not obtain consumer connection no. 2: ", err)
			}
			consumerConn3, err := net.Dial("tcp", ":8044")
			if err != nil || consumerConn3 == nil {
				t.Error("could not obtain consumer connection no. 3: ", err)
			}

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
				checkOutputOnConsumerConnection(cc, tc.payload1, t)
			}

			if _, err := producerConn.Write(tc.payload2); err != nil {
				t.Error("could not write payload to producer: ", err)
			}

			for _, cc := range consumerConns {
				checkOutputOnConsumerConnection(cc, tc.payload2, t)
			}
		})
	}
}
