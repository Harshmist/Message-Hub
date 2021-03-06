package main

import (
	"bufio"
	"expvar"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	logger           *log.Logger
	startTime        time.Time
	allUsers         []user
	rooms                      = make([][]user, 3)
	roomsHistory               = make([][]string, 3)
	categories       [3]string = [3]string{"Dogs", "Cats", "Dolphins"}
	roomZeroChan               = make(chan [3]string)
	roomOneChan                = make(chan [3]string)
	roomTwoChan                = make(chan [3]string)
	roomZeroHistChan           = make(chan string)
	roomOneHistChan            = make(chan string)
	roomTwoHistChan            = make(chan string)
	joinChan                   = make(chan net.Conn)
	requestsMonitor            = expvar.NewInt("Total Requests")
	invalidRequests            = expvar.NewInt("Total invalid requests")
	totalUsers                 = expvar.NewInt(("Total Users"))
)

type user struct {
	name    string
	address net.Conn
}

func init() {
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	logger = log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)

	//initializing slices for rooms and room chat history
	for i := 0; i < 3; i++ {
		rooms[i] = make([]user, 0)
		roomsHistory[i] = make([]string, 0)
	}
}

func startTCP() {
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Println("user failed to connect")
		}
		logger.Printf("User %v connected\n", conn)
		io.WriteString(conn, "Welcome to the message hub!\n Write [CMD] for a list of commands\n")
		joinChan <- conn

		go handler(conn)

	}
}

func userJoin() {
	var newUser user
	for {
		select {
		case newAddress := <-joinChan:
			newUser.address = newAddress
			allUsers = append(allUsers, newUser)
			for _, user := range allUsers {
				io.WriteString(user.address, fmt.Sprintf("User %v has joined the server\n", newAddress))
			}
		}
	}

}

func handler(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		switch fields[0] {
		case "CMD":
			io.WriteString(conn, "LIST: Shows list of categories\nMSG (category number): Sends message to category subscribers\nSUB(category number): subscribes you to that category\nNICK (name): Changes your nickname\nHIST (category number): Shows last 30 messages in this channel\n")
		case "LIST":
			io.WriteString(conn, "List of categories are:\n")
			for k, v := range categories {
				io.WriteString(conn, fmt.Sprintf("%v: %v\n", k, v))
			}
		case "SUB":
			var user user
			user.address = conn
			subChannel := fields[1]
			switch subChannel {
			case "0":
				rooms[0] = append(rooms[0], user)
				io.WriteString(conn, fmt.Sprintf("Now subscribed to channel %v!\n", fields[1]))
			case "1":
				rooms[1] = append(rooms[1], user)
				io.WriteString(conn, fmt.Sprintf("Now subscribed to channel %v!\n", fields[1]))
			case "2":
				rooms[2] = append(rooms[2], user)
				io.WriteString(conn, fmt.Sprintf("Now subscribed to channel %v!\n", fields[1]))
			default:
				io.WriteString(conn, fmt.Sprintf("channel %v does not exist. Try the [LIST] command to see what channels are open.\n", subChannel))
			}

		case "NICK":
			newName := fields[1]
			for i := 0; i < len(allUsers); i++ {
				if allUsers[i].address == conn {
					allUsers[i].name = newName
					io.WriteString(conn, "Your new name is "+newName+"\n")
					for _, user := range allUsers {
						if user.address != conn {
							io.WriteString(user.address, fmt.Sprintf("User %v has changed their nickname to %v\n", conn, newName))
						}
					}
				}
			}
		case "HIST":
			subNum := fields[1]
			switch subNum {
			case "0":
				if len(rooms[0]) < 30 {
					for _, v := range roomsHistory[0] {
						io.WriteString(conn, fmt.Sprintf("%v\n", v))
					}
				} else {
					for i := len(rooms[0]) - 30; i < len(rooms[0]); i++ {
						io.WriteString(conn, fmt.Sprintf("%v\n", rooms[0][i]))
					}
				}

			case "1":
				if len(rooms[1]) < 30 {
					for _, v := range roomsHistory[1] {
						io.WriteString(conn, fmt.Sprintf("%v\n", v))
					}
				} else {
					for i := len(rooms[0]) - 30; i < len(rooms[1]); i++ {
						io.WriteString(conn, fmt.Sprintf("%v\n", rooms[1][i]))
					}
				}
			case "2":
				if len(rooms[2]) < 30 {
					for _, v := range roomsHistory[2] {
						io.WriteString(conn, fmt.Sprintf("%v\n", v))
					}
				} else {
					for i := len(rooms[0]) - 30; i < len(rooms[2]); i++ {
						io.WriteString(conn, fmt.Sprintf("%v\n", rooms[2][i]))
					}
				}
			}

		case "MSG":
			var msgArr [3]string
			func() {
				for i := 0; i < len(allUsers); i++ {
					if conn == allUsers[i].address {
						switch allUsers[i].name {
						case "":
							msgArr[0] = conn.LocalAddr().String()
						default:
							msgArr[0] = allUsers[i].name
						}

					}
				}
			}()
			subNum := fields[1]
			message := strings.Join(fields[2:], " ")
			msgArr[1] = subNum
			msgArr[2] = message

			switch fields[1] {
			case "0":
				roomZeroChan <- msgArr

			case "1":
				roomOneChan <- msgArr
			case "2":
				roomTwoChan <- msgArr
			default:
				io.WriteString(conn, fmt.Sprintf("channel %v does not exist. Try the [LIST] command to see what channels are open.\n", subNum))
			}

		}
	}
}

func historyStorage() {
	for {
		select {
		case msg := <-roomZeroHistChan:
			roomsHistory[0] = append(roomsHistory[0], msg)
		case msg := <-roomOneHistChan:
			roomsHistory[1] = append(roomsHistory[1], msg)
		case msg := <-roomTwoHistChan:
			roomsHistory[2] = append(roomsHistory[2], msg)
		}
	}
}

func msgBroadcast() {
	for {
		select {

		case msg0 := <-roomZeroChan:
			msgString := fmt.Sprintf("%v wrote on channel %v: %v\n", msg0[0], msg0[1], msg0[2])
			roomZeroHistChan <- msgString
			for _, v := range rooms[0] {
				conn := v.address
				io.WriteString(conn, msgString)
			}
		case msg1 := <-roomOneChan:
			msgString := fmt.Sprintf("%v wrote on channel %v: %v\n", msg1[0], msg1[1], msg1[2])
			roomOneHistChan <- msgString
			for _, v := range rooms[1] {
				conn := v.address
				io.WriteString(conn, msgString)
			}
		case msg2 := <-roomTwoChan:
			msgString := fmt.Sprintf("%v wrote on channel %v: %v\n", msg2[0], msg2[1], msg2[2])
			roomTwoHistChan <- msgString
			for _, v := range rooms[2] {
				conn := v.address
				io.WriteString(conn, msgString)
			}
		}
	}
}

func main() {
	go startTCP()
	go msgBroadcast()
	go historyStorage()
	go userJoin()

	fmt.Scanln()

}
