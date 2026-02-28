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

var (
	serverAddr string
	userName   string
	gameState  = "Ожидание..."
	chatBox    = ""
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Адрес сервера (https://turbo-orbit-4jw7r4795jxjf5qrj-8080.app.github.dev/): ")
	scanner.Scan()
	serverAddr = scanner.Text()

	fmt.Print("Ваше имя: ")
	scanner.Scan()
	userName = scanner.Text()

	// 1. Поток обновления данных
	go func() {
		for {
			resp, err := http.Get(serverAddr)
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				parts := strings.Split(string(body), "|||")
				if len(parts) == 2 {
					gameState = parts[0]
					chatBox = parts[1]
				}
				resp.Body.Close()
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// 2. Поток отрисовки интерфейса
	go func() {
		for {
			// Очистка экрана (ANSI escape codes)
			fmt.Print("\033[H\033[2J") 
			fmt.Printf("=== ИГРОК: %s ===\n", userName)
			fmt.Printf("СОСТОЯНИЕ: %s\n", gameState)
			fmt.Println("---------------------------------")
			fmt.Println("ЧАТ:")
			fmt.Println(chatBox)
			fmt.Println("---------------------------------")
			fmt.Print("Введите (head/body/legs) или сообщение: ")
			time.Sleep(1 * time.Second)
		}
	}()

	// Регистрация
	http.Post(serverAddr, "text/plain", strings.NewReader("register="+userName))

	// 3. Основной цикл ввода
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" { continue }

		// Если это боевая команда
		if input == "head" || input == "body" || input == "legs" {
			if strings.Contains(gameState, "ATTACK") {
				http.Post(serverAddr, "text/plain", strings.NewReader("attack="+userName+":"+input))
			} else if strings.Contains(gameState, "DEFENSE") {
				http.Post(serverAddr, "text/plain", strings.NewReader("defense="+userName+":"+input))
			}
		} else {
			// Иначе отправляем в чат
			msg := "[CHAT]" + userName + ": " + input
			http.Post(serverAddr, "text/plain", strings.NewReader(msg))
		}
	}
}

