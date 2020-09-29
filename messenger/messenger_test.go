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
	msgr = NewMessenger(8044)
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

func setupAndTestConnection(port int, connType string, name string, t *testing.T) net.Conn {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil || conn == nil {
		t.Error("could not obtain connection: ", err)
	}
	checkOutput(conn, []byte("Type `c` for `consumer`, `p` for producer or `chat` for chat mode\n"), t)
	if _, err := conn.Write([]byte(fmt.Sprintf("%s\n", connType))); err != nil {
		t.Error("could not write payload to server: ", err)
	}
	switch connType {
	case "p":
		checkOutput(conn, []byte("Entering `producer` mode\n"), t)
	case "c":
		checkOutput(conn, []byte("Entering `consumer` mode\n"), t)
	case "chat":
		checkOutput(conn, []byte("Entering `chat` mode, enter your name:\n"), t)
		if _, err := conn.Write([]byte(fmt.Sprintf("%s\n", name))); err != nil {
			t.Error("could not write payload to server: ", err)
		}
	default:
		t.Error("bad test case, unknown connType provided: ", connType)
	}
	return conn
}

func getExpectedChatMessage(name string, message []byte) []byte {
	return []byte(fmt.Sprintf("%s: %s", name, message))
}

func TestMessenger_Run(t *testing.T) {
	time.Sleep(1 * time.Second) // Give time for messenger to spin up.
	setupAndTestConnection(8044, "c", "", t)
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

	producerConn := setupAndTestConnection(8044, "p", "", t)
	consumerConn := setupAndTestConnection(8044, "c", "", t)

	defer producerConn.Close()
	defer consumerConn.Close()
	for _, tc := range tcs {
		t.Run(tc.test, func(t *testing.T) {
			if _, err := producerConn.Write(tc.payload); err != nil {
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

	producerConn := setupAndTestConnection(8044, "p", "", t)
	producerConn2 := setupAndTestConnection(8044, "p", "", t)
	consumerConn := setupAndTestConnection(8044, "c", "", t)
	consumerConn2 := setupAndTestConnection(8044, "c", "", t)
	consumerConn3 := setupAndTestConnection(8044, "c", "", t)

	defer producerConn.Close()
	defer producerConn2.Close()
	defer consumerConn.Close()
	defer consumerConn2.Close()
	defer consumerConn3.Close()

	consumerConns := []net.Conn{consumerConn, consumerConn2, consumerConn3}
	for _, tc := range tcs {
		t.Run(tc.test, func(t *testing.T) {
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

func TestMessengerWithMultipleChatUsers(t *testing.T) {
	tcs := []struct {
		test     string
		user1    string
		payload1 []byte
		user2    string
		payload2 []byte
	}{
		{
			"Sending a msg via any producer connection, sends the same message to all consumer connections",
			"obi-van-kenobi",
			[]byte("Hello there\n"),
			"gen. grievious",
			[]byte("General Kenobi\n"),
		},
		{
			"Sending a msg via any producer connection, sends the same message to all consumer connections, part deux",
			"InigoMontoya_nr1",
			[]byte("Hello. My name is Inigo Montoya. You killed my father. Prepare to die.\n"),
			"fingers_6",
			[]byte("Huh?\n"),
		},
	}

	time.Sleep(1 * time.Second)

	for _, tc := range tcs {
		t.Run(tc.test, func(t *testing.T) {
			chatUser1 := setupAndTestConnection(8044, "chat", tc.user1, t)
			chatUser2 := setupAndTestConnection(8044, "chat", tc.user2, t)
			if _, err := chatUser1.Write(tc.payload1); err != nil {
				t.Error("could not write payload to producer: ", err)
			}

			checkOutput(chatUser2, getExpectedChatMessage(tc.user1, tc.payload1), t)

			if _, err := chatUser2.Write(tc.payload2); err != nil {
				t.Error("could not write payload to producer: ", err)
			}

			checkOutput(chatUser1, getExpectedChatMessage(tc.user2, tc.payload2), t)
			chatUser1.Close()
			chatUser2.Close()
		})
	}
}
