package goesl

// Set - Helper func to execute SET application against active ESL session
func (sc *SocketConnection) ExecuteSet(key, value string, sync bool) (m *Message, err error) {
	return sc.Execute("set", key+"="+value, sync)
}

// ExecuteAnswer - Helper func to help with executing Answer against active ESL session
func (sc *SocketConnection) ExecuteAnswer(args string, sync bool) (m *Message, err error) {
	return sc.Execute("answer", args, sync)
}

// ExecuteHangup - Helper func to help with executing Hangup against active ESL session
func (sc *SocketConnection) ExecuteHangup(uuid, args string, sync bool) (m *Message, err error) {
	if uuid != "" {
		return sc.ExecuteUUID(uuid, "hangup", args, sync)
	}

	return sc.Execute("hangup", args, sync)
}

// Api - Helper to attach api in front of the command
func (sc *SocketConnection) Api(command string) error {
	return sc.Send("api " + command)
}

// BgApi - Helper to attach bgapi in front of the command
func (sc *SocketConnection) BgApi(command string) error {
	return sc.Send("bgapi " + command)
}

// Connect - Helper function to handle connection. Each outbound server when handling needs to connect e.g. accept
// connection in order to do answer, hangup etc.
func (sc *SocketConnection) Connect() error {
	return sc.Send("connect")
}

// Exit - Used to send exit signal to ESL> Will hangup call and close connection
func (sc *SocketConnection) Exit() error {
	return sc.Send("exit")
}
