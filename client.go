package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "legendary-robot-4jw7r47957p435jv-8080.app.github.dev:8080")
	if err != nil {
		log.Println("Ошибка соединения:", err)
		return
	}
	defer conn.Close()

	messages := make(chan string, 5)
	go receiveMessages(conn, messages)
	go func() {
		for msg := range messages {
			fmt.Println("Сообщение:", msg)
		}
	}()
	SendMessage(conn)
}

func SendMessage(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(conn)
	for {
		fmt.Print("Введите сообщение: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		_, err := writer.WriteString(input + "\n")
		if err != nil {
			log.Println("Ошибка отправки:", err)
			break
		}
		writer.Flush()
	}
}

func receiveMessages(conn net.Conn, messages chan string) {
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Соединение разорвано:", err)
			close(messages)
			return
		}
		messages <- strings.TrimSpace(msg)
	}
}