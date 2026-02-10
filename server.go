package main

import (
	"fmt"
	"log"
	"net"
	"bufio"
	"strings"
)

type Client struct {
	conn net.Conn
	name string
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:8080") // для публичного доступа
	if err != nil {
		log.Println("Ошибка запуска сервера:", err)
		return
	}
	defer l.Close()
	fmt.Println("Сервер запущен на 0.0.0.0:8080")

	clients := make(map[net.Conn]*Client)
	messages := make(chan string, 10)
	go printMessages(messages, clients)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Ошибка при соединении:", err)
			continue
		}
		client := &Client{conn: conn}
		clients[conn] = client
		go handleUserConnection(client, messages, clients)
	}
}

func printMessages(messages <-chan string, clients map[net.Conn]*Client) {
	for msg := range messages {
		fmt.Println("Сообщение клиентам:", msg)
		for _, client := range clients {
			writer := bufio.NewWriter(client.conn)
			writer.WriteString(msg + "\n")
			writer.Flush()
		}
	}
}

func handleUserConnection(client *Client, messages chan string, clients map[net.Conn]*Client) {
	defer func() {
		client.conn.Close()
		delete(clients, client.conn)
	}()
	reader := bufio.NewReader(client.conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Клиент отключился:", err)
			return
		}
		msg = strings.TrimSpace(msg)
		if msg == "" {
			continue
		}
		messages <- fmt.Sprintf("Клиент: %s", msg)
	}
}