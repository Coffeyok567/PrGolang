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

	// Ввод имени
	fmt.Print("Введите имя игрока: ")
	scanner.Scan()
	name := scanner.Text()

	// Регистрация
	http.Post(server, "text/plain",
		strings.NewReader("register="+name))

	for {

		// ===== АТАКА =====
		fmt.Print("Выберите атаку (head/body/legs): ")
		scanner.Scan()
		attack := scanner.Text()

		http.Post(server, "text/plain",
			strings.NewReader("attack="+name+":"+attack))

		time.Sleep(2 * time.Second)

		// ===== ЗАЩИТА =====
		fmt.Print("Выберите защиту (head/body/legs): ")
		scanner.Scan()
		defense := scanner.Text()

		http.Post(server, "text/plain",
			strings.NewReader("defense="+name+":"+defense))

		time.Sleep(2 * time.Second)

		// ===== РЕЗУЛЬТАТ =====
		resp, _ := http.Get(server)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if string(body) != "" {
			fmt.Println("\n" + string(body))
		}
	}
}
