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

var display_chan = make(chan string, 10)

func main() {
	server := "http://localhost:8080"
	scanner := bufio.NewScanner(os.Stdin)

	// Горутина для печати сообщений из канала
	go func() {
		for text := range display_chan {
			fmt.Println(text)
		}
	}()

	// Горутина для периодического запроса истории чата
	go func() {
		last_count := 0

		for {
			resp, err := http.Get(server)
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				lines := strings.Split(strings.TrimSpace(string(body)), "\n")

				if len(lines) > last_count && lines[0] != "" {
					for i := last_count; i < len(lines); i++ {
						if lines[i] != "" {
							display_chan <- lines[i]
						}
					}
					last_count = len(lines)
				}
				resp.Body.Close()
			}
			time.Sleep(1 * time.Second)
		}
	}()

	// Регистрация в PVP
	fmt.Print("Введите имя для PVP: ")
	scanner.Scan()
	name := scanner.Text()

	// Регистрируемся на сервере
	resp, err := http.Post(server, "text/plain", strings.NewReader("register="+name))
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		if string(body) == "SERVER_FULL" {
			fmt.Println("❌ Сервер PVP полон! Будете просто общаться в чате.")
		} else {
			fmt.Println("✅ Зарегистрированы в PVP!")
		}
		resp.Body.Close()
	}

	// Отдельная горутина для игрового ввода
	go func() {
		gameScanner := bufio.NewScanner(os.Stdin)
		lastPhase := ""

		for {
			// Проверяем фазу игры
			resp, err := http.Get(server)
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				lines := strings.Split(string(body), "\n")
				
				// Ищем информацию о фазе в последних строках
				for _, line := range lines {
					if strings.Contains(line, "Фаза:") {
						parts := strings.Split(line, ":")
						if len(parts) > 1 {
							currentPhase := strings.TrimSpace(parts[1])
							
							if currentPhase != lastPhase {
								switch currentPhase {
								case "ATTACK":
									fmt.Print("\n⚔️ Фаза АТАКИ (head/body/legs): ")
									gameScanner.Scan()
									attack := gameScanner.Text()
									http.Post(server, "text/plain", strings.NewReader("attack="+name+":"+attack))
									
								case "DEFENSE":
									fmt.Print("\n🛡️ Фаза ЗАЩИТЫ (head/body/legs): ")
									gameScanner.Scan()
									defense := gameScanner.Text()
									http.Post(server, "text/plain", strings.NewReader("defense="+name+":"+defense))
								}
								lastPhase = currentPhase
							}
						}
					}
				}
				resp.Body.Close()
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	// Основной цикл для отправки сообщений в чат
	fmt.Println("\n📝 Введите сообщение в чат (или 'exit' для выхода):")
	for scanner.Scan() {
		message := scanner.Text()
		
		if message == "exit" {
			break
		}

		if message != "" {
			full_message := "[" + name + "]: " + message
			http.Post(server, "text/plain", strings.NewReader(full_message))
		}
	}
}
