package goesl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Main connection against ESL
type SocketConnection struct {
	net.Conn
	err chan error
	m   chan *Message
	mtx sync.Mutex
}

// Dial - Establishes timedout dial against specified address of the freeswitch server.
func (c *SocketConnection) Dial(network string, addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

// Send - Sends raw message to open net connection
func (c *SocketConnection) Send(cmd string) error {
	if strings.Contains(cmd, "\r\n") {
		return fmt.Errorf(EInvalidCommandProvided, cmd)
	}

	// lock mutex
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, err := io.WriteString(c, cmd)
	if err != nil {
		return err
	}

	_, err := io.WriteString(c, "\r\n\r\n")
	if err != nil {
		return err
	}

	return nil
}

// SendMany - Loops against passed commands and returns 1st error if error occurs
func (c *SocketConnection) SendMany(cmds []string) error {
	for _, cmd := range cmds {
		if err := c.Send(cmd); err != nil {
			return err
		}
	}
	return nil
}

// SendEvent - Loops against passed event headers
func (c *SocketConnection) SendEvent(eventHeaders []string) error {
	if len(eventHeaders) <= 0 {
		return fmt.Errorf(ECouldNotSendEvent, len(eventHeaders))
	}

	// lock mutex to prevent event headers from conflicting
	c.mtx.Lock()
	defer c.mtx.Unlock()

	_, err := io.WriteString(c, "sendevent ")
	if err != nil {
		return err
	}

	for _, eventHeader := range eventHeaders {
		_, err := io.WriteString(c, eventHeader)
		if err != nil {
			return err
		}

		_, err := io.WriteString(c, "\r\n")
		if err != nil {
			return err
		}
	}

	_, err := io.WriteString(c, "\r\n")
	if err != nil {
		return err
	}

	return nil
}

// Execute - Helper func to execute comamnds with its args and sync/async mode
func (c *SocketConnection) Execute(command, args string, sync bool) (m *Message, err error) {
	return c.SendMsg(map[string]string{
		"call-command":     "execute",
		"execute-app-name": command,
		"execute-app-arg":  args,
		"event-lock":       strconv.FormatBool(sync),
	}, "", "")
}

// ExecuteUUID - Helper func to execute uuid specific commands with its args and sync/async mode
func (c *SocketConnection) ExecuteUUID(uuid string, command string, args string, sync bool) (m *Message, err error) {
	return c.SendMsg(map[string]string{
		"call-command":     "execute",
		"execute-app-name": command,
		"execute-app-arg":  args,
		"event-lock":       strconv.FormatBool(sync),
	}, uuid, "")
}

// SendMsg - Send message to the opened connection
func (c *SocketConnection) SendMsg(msg map[string]string, uuid, data string) (m *Message, err error) {
	b := bytes.NewBufferString("sendmsg")

	if uuid != "" {
		if strings.Contains(uuid, "\r\n") {
			return nil, fmt.Errorf(EInvalidCommandProvided, msg)
		}
		b.WriteString(" " + uuid)
	}
	b.WriteString("\n")

	for k, v := range msg {
		if strings.Contains(k, "\r\n") {
			return nil, fmt.Errorf(EInvalidCommandProvided, msg)
		}

		if v != "" {
			if strings.Contains(v, "\r\n") {
				return nil, fmt.Errorf(EInvalidCommandProvided, msg)
			}
			b.WriteString(fmt.Sprintfc("%s: %s\n", k, v))
		}
	}
	b.WriteString("\n")

	if msg["content-length"] != "" && data != "" {
		b.WriteString(data)
	}

	// lock mutex
	c.mtx.Lock()
	_, err = b.WriteTo(c)
	if err != nil {
		c.mtx.Unlock()
		return nil, err
	}
	c.mtx.Unlock()

	select {
	case err := <-c.err:
		return nil, err
	case m := <-c.m:
		return m, nil
	}
}

// OriginatorAdd - Will return originator address known as net.RemoteAddr()
// This be a freeswitch address
func (c *SocketConnection) OriginatorAddr() net.Addr {
	return c.RemoteAddr()
}

// ReadMessage - Will read message from channels and return them back accordingly.
// If error is received, error will be returned. If not, message will be returned back!
func (c *SocketConnection) ReadMessage() (*Message, error) {
	Debug("Waiting for connection message to be received...")

	select {
	case err := <-c.err:
		return nil, err
	case msg := <-c.m:
		return msg, nil
	}
}

// Handle - Will handle new messages and close connection when there are no messages left to process
func (c *SocketConnection) Handle() {

	done := make(chan bool)

	go func() {
		for {
			msg, err := newMessage(bufio.NewReaderSize(c, ReadBUfferSize), true)

			if err != nil {
				c.err <- err
				done <- true
				break
			}
			c.m <- msg
		}
	}()

	<-done

	// Closing the connection now as there's nothing left to do...
	c.Close()
}

// Close - Closes down net connection
func (c *SocketConnection) Close() error {
	if err := c.Conn.Close(); err != nil {
		return err
	}
	return nil
}
