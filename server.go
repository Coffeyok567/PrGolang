package main

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "sync"
)

type Client struct {
    conn net.Conn
    name string
}

var (
    clients   = make(map[*Client]bool)
    broadcast = make(chan string, 10) // канал на 10 сообщений
    mutex     = &sync.Mutex{}
)

func main() {
    listener, err := net.Listen("tcp", ":8080")
    if err != nil {
        log.Fatal(err)
    }
    defer listener.Close()

    fmt.Println("Сервер запущен на порту 8080")

    // Горутина для рассылки сообщений всем клиентам
    go broadcaster()

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Println(err)
            continue
        }
        go handleClient(conn)
    }
}

func handleClient(conn net.Conn) {
    defer conn.Close()

    client := &Client{conn: conn}
    mutex.Lock()
    clients[client] = true
    mutex.Unlock()

    // Запрос имени
    conn.Write([]byte("Введите ваше имя: "))
    scanner := bufio.NewScanner(conn)
    scanner.Scan()
    client.name = scanner.Text()

    welcome := fmt.Sprintf("%s присоединился к чату.\n", client.name)
    broadcast <- welcome

    // Приём сообщений от клиента
    for scanner.Scan() {
        msg := scanner.Text()
        fullMsg := fmt.Sprintf("%s: %s\n", client.name, msg)
        broadcast <- fullMsg
    }

    // Клиент отключился
    mutex.Lock()
    delete(clients, client)
    mutex.Unlock()
    leaveMsg := fmt.Sprintf("%s покинул чат.\n", client.name)
    broadcast <- leaveMsg
}

func broadcaster() {
    for msg := range broadcast {
        mutex.Lock()
        for client := range clients {
            _, err := client.conn.Write([]byte(msg))
            if err != nil {
                client.conn.Close()
                delete(clients, client)
            }
        }
        mutex.Unlock()
    }
}