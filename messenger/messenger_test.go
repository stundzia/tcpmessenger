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
	tt := []struct {
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
	for _, tc := range tt {
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

			out := make([]byte, 128)
			if _, err := consumerConn.Read(out); err == nil {
				out = bytes.Trim(out, "\x00")
				if bytes.Compare(out, tc.payload) != 0 {
					t.Errorf("response did not match expected output - got `%s`, but expected `%s`", string(out), string(tc.payload))
				}
			} else {
				t.Error("could not read from connection")
			}
		})
	}
}

func TestMessengerWithMultipleProducersAndConsumers(t *testing.T) {
	tt := []struct {
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
	for _, tc := range tt {
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
				out := make([]byte, 128)
				if _, err := cc.Read(out); err == nil {
					out = bytes.Trim(out, "\x00")
					if bytes.Compare(out, tc.payload1) != 0 {
						t.Errorf("response did not match expected output - got `%s`, but expected `%s`", string(out), string(tc.payload1))
					}
				} else {
					t.Error("could not read from connection")
				}
			}

			if _, err := producerConn.Write(tc.payload2); err != nil {
				t.Error("could not write payload to producer: ", err)
			}

			for _, cc := range consumerConns {
				out := make([]byte, 128)
				if _, err := cc.Read(out); err == nil {
					out = bytes.Trim(out, "\x00")
					if bytes.Compare(out, tc.payload2) != 0 {
						t.Errorf("response did not match expected output - got `%s`, but expected `%s`", string(out), string(tc.payload2))
					}
				} else {
					t.Error("could not read from connection")
				}
			}
		})
	}
}
