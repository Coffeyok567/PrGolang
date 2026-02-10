package main

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "os"
    "sync"
)

func main() {
    var serverAddr string
    fmt.Print("Введите адрес сервера (например, 192.168.1.10:8080): ")
    fmt.Scanln(&serverAddr)

    conn, err := net.Dial("tcp", serverAddr)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    fmt.Println("Подключено к серверу. Введите сообщения.")

    var wg sync.WaitGroup
    wg.Add(2)

    // Горутина для чтения сообщений от сервера
    go func() {
        defer wg.Done()
        scanner := bufio.NewScanner(conn)
        for scanner.Scan() {
            fmt.Println(scanner.Text())
        }
    }()

    // Горутина для отправки сообщений на сервер
    go func() {
        defer wg.Done()
        scanner := bufio.NewScanner(os.Stdin)
        for scanner.Scan() {
            msg := scanner.Text()
            _, err := fmt.Fprintln(conn, msg)
            if err != nil {
                log.Println(err)
                break
            }
        }
    }()

    wg.Wait()
}