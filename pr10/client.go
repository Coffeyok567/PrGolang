package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {

	server := "http://localhost:8080"
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Введите имя: ")
	scanner.Scan()
	name := scanner.Text()

	http.Post(server, "text/plain",
		strings.NewReader("register="+name))

	for {

		resp, _ := http.Get(server)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		state := string(body)

		switch state {

		case "WAIT":
			fmt.Println("Ожидание второго игрока или его хода...")
			time.Sleep(2 * time.Second)

		case "ATTACK":
			fmt.Print("Выберите атаку (head/body/legs): ")
			scanner.Scan()
			attack := scanner.Text()
			http.Post(server, "text/plain",
				strings.NewReader("attack="+name+":"+attack))

		case "DEFENSE":
			fmt.Print("Выберите защиту (head/body/legs): ")
			scanner.Scan()
			defense := scanner.Text()
			http.Post(server, "text/plain",
				strings.NewReader("defense="+name+":"+defense))

		default:
			fmt.Println("\n" + state)
			time.Sleep(3 * time.Second)
		}
	}
}
