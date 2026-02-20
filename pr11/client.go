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

var lastChatCount = 0

func main() {
	server := "http://localhost:8080"
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Введите ваш ник: ")
	scanner.Scan()
	name := scanner.Text()

	// Регистрация
	http.Post(server, "text/plain", strings.NewReader("register="+name))

	// Горутина для обновления чата и состояния игры
	go func() {
		for {
			resp, err := http.Get(server)
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				parts := strings.Split(string(body), "|---CHAT---|")
				gameStatus := parts[0]

				// Если есть новые сообщения в чате, печатаем их
				if len(parts) > 1 {
					chatLines := strings.Split(parts[1], "\n")
					if len(chatLines) > lastChatCount && chatLines[0] != "" {
						for i := lastChatCount; i < len(chatLines); i++ {
							if chatLines[i] != "" {
								fmt.Printf("\n[ЧАТ] %s\n> ", chatLines[i])
							}
						}
						lastChatCount = len(chatLines)
					}
				}

				// Вывод состояния игры (только если это не ожидание)
				if gameStatus != "WAIT" && gameStatus != "ATTACK" && gameStatus != "DEFENSE" {
					fmt.Printf("\n%s\n> ", gameStatus)
				}
			}
			time.Sleep(2 * time.Second)
		}
	}()

	fmt.Println("Инструкция: Просто пишите head/body/legs для игры.")
	fmt.Println("Чтобы написать в чат, начните сообщение с '/' (например: /привет!)")

	for {
		fmt.Print("> ")
		scanner.Scan()
		input := scanner.Text()

		if strings.HasPrefix(input, "/") {
			// Отправка в чат
			msg := fmt.Sprintf("[CHAT][%s]: %s", name, strings.TrimPrefix(input, "/"))
			http.Post(server, "text/plain", strings.NewReader(msg))
		} else {
			// Игровое действие (автоматически определяем фазу по запросу к серверу)
			// Для простоты отправляем и как атаку, и как защиту — сервер сам поймет по фазе
			http.Post(server, "text/plain", strings.NewReader("attack="+name+":"+input))
			http.Post(server, "text/plain", strings.NewReader("defense="+name+":"+input))
		}
	}
}
