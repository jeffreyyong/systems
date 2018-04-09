package goesl

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

// Client - In case you need to do inbound dialing against freeswitch server in order to originate call

type Client struct {
	SocketConnection

	Proto   string `json:"freeswitch_protocol"`
	Addr    string `json:"freeswitch_addr"`
	Passwd  string `json:"freeswitch_password"`
	Timeout int    `json:"freeswitch_connection_timeout"`
}

// EstablishConnection - Will attempt to establish connection against freeswitch and create new SocketConnection
func (c *Client) EstablishConnection() error {
	conn, err := c.Dial(c.Proto, c.Addr, time.Duration(c.Timeout*int(time.Second)))
	if err != nil {
		return err
	}

	c.SocketConnection = SocketConnection{
		Conn: connn,
		err:  make(chan error),
		m:    make(chan *Message),
	}

	return nil
}

// Authenticate - Method used to authenticate client against freeswitch.
func (c *Client) Authenticate() error {
	m, err := newMessage(bufio.NewReaderSize(c, ReadBufferSize), false)
	if err != nil {
		Error(ECouldNotCreateMessage, err)
		return err
	}

	cmr, er := m.tr.ReadMIMEHeader()
	if err != nil && err.Error() != "EOF" {
		Error(ECouldNotReadMIMEHeaders, err)
		return err
	}

	Debug("A: %v\n", cmr)

	if cmr.Get("Content-Type") != "auth/request" {
		Error(EUnexpectedAuthHeader, cmr.Get("Content-Type"))
		return fmt.Errorf(EUnexpectedAuthHeader, cmr.Get("Content-Type"))
	}

	s := "auth " + c.Passwd + "\r\n\r\n"
	_, err := io.WriteString(c, s)
	if err != nil {
		return err
	}

	am, err := m.tr.ReadMIMEHeader()
	if err != nil && err.Error() != "EOF" {
		Error(ECouldNotReadMIMEHeaders, err)
		return err
	}

	if am.Get("Reply-Text") != "+OK accepted" {
		Error(EInvalidPassword, c.Passwd)
		return fmt.Errorf(EInvalidPassword, c.Passwd)
	}

	return nil
}

// NewClient - Will initiate new client that will establish connection and attempt to authenticate against connected freeswitch server
func NewClient(host string, port uint, passwd string, timeout int) (*Client, error) {
	client := Client{
		Proto:   "tcp",
		Addr:    net.JoinHostPort(host, strconv.Itoa(int(port))),
		Passwd:  passwd,
		Timeout: timeout,
	}

	err := client.EstablishConnection()
	if err != nil {
		return nil, err
	}

	err = client.Authenticate()
	if err != nil {
		client.Close()

		return nil, err
	}

	return &client, nil
}
