package main

import (
	"bufio"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net"
	"time"
)

var (
	enteringChannel = make(chan *User)
	leavingChannel  = make(chan *User)
	messageChannel  = make(chan Message, 8)
)

type User struct {
	UUID           uuid.UUID
	Addr           string
	EnterAt        time.Time
	MessageChannel chan string
}

type Message struct {
	OwnerID uuid.UUID
	Content string
}

func main() {
	listener, err := net.Listen("tcp", ":2020")
	if err != nil {
		panic(any(err))
	}

	go broadcaster()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go handleConn(conn)
	}
}

func broadcaster() {
	users := make(map[*User]struct{})

	for {
		select {
		case user := <-enteringChannel:
			users[user] = struct{}{}
		case user := <-leavingChannel:
			delete(users, user)
			close(user.MessageChannel)
		case msg := <-messageChannel:
			for user := range users {
				if user.UUID == msg.OwnerID {
					continue
				}
				user.MessageChannel <- msg.Content
			}
		}
	}
}

func sendMessage(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg)
	}
}

func GenUserID() uuid.UUID {
	return uuid.New()
}

func handleConn(conn net.Conn) {
	user := &User{
		UUID:           GenUserID(),
		Addr:           conn.RemoteAddr().String(),
		EnterAt:        time.Now(),
		MessageChannel: make(chan string, 8),
	}

	go sendMessage(conn, user.MessageChannel)

	user.MessageChannel <- "Welcome, " + user.UUID.String()
	messageChannel <- Message{
		OwnerID: user.UUID,
		Content: "user:'" + user.UUID.String() + "'has enter",
	}

	enteringChannel <- user

	input := bufio.NewScanner(conn)
	for input.Scan() {
		messageChannel <- Message{
			OwnerID: user.UUID,
			Content: user.UUID.String() + ":" + input.Text(),
		}
	}

	if err := input.Err(); err != nil {
		log.Println("读取错误", err)
	}

	leavingChannel <- user
	messageChannel <- Message{
		OwnerID: user.UUID,
		Content: "user:`" + user.UUID.String() + "` has left",
	}
}
