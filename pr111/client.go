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
	// Запрашиваем URL сервера
	fmt.Print("Введите URL сервера (например: https://ваш-код-8080.app.github.dev): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	server := strings.TrimSpace(scanner.Text())
	
	// Запрос имени
	fmt.Print("Введите ваше имя: ")
	scanner.Scan()
	name := scanner.Text()
	
	// Регистрация в игре
	fmt.Print("Хотите участвовать в PvP? (да/нет): ")
	scanner.Scan()
	if strings.ToLower(scanner.Text()) == "да" {
		resp, err := http.Post(server, "text/plain", 
			strings.NewReader("register="+name))
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			if string(body) == "SERVER_FULL" {
				fmt.Println("❌ Сервер полон (максимум 2 игрока)")
			} else {
				fmt.Println("✅ Вы зарегистрированы в PvP режиме!")
			}
			resp.Body.Close()
		}
	}
	
	// Горутина для получения обновлений
	go func() {
		lastMsgCount := 0
		
		for {
			resp, err := http.Get(server)
			if err == nil {
				body, _ := io.ReadAll(resp.Body)
				lines := strings.Split(string(body), "\n")
				
				if len(lines) > lastMsgCount {
					for i := lastMsgCount; i < len(lines); i++ {
						if lines[i] != "" {
							fmt.Println(lines[i])
						}
					}
					lastMsgCount = len(lines)
				}
				resp.Body.Close()
			}
			time.Sleep(2 * time.Second)
		}
	}()
	
	// Основной цикл ввода
	fmt.Println("\n💬 Введите сообщение или игровую команду:")
	fmt.Println("🎮 Игровые команды: /attack head/body/legs, /defense head/body/legs")
	fmt.Println("📝 Обычный текст - сообщение в чат")
	
	for scanner.Scan() {
		text := scanner.Text()
		
		if strings.HasPrefix(text, "/attack ") {
			part := strings.TrimPrefix(text, "/attack ")
			resp, err := http.Post(server, "text/plain",
				strings.NewReader("attack="+name+":"+part))
			if err == nil {
				resp.Body.Close()
			}
		} else if strings.HasPrefix(text, "/defense ") {
			part := strings.TrimPrefix(text, "/defense ")
			resp, err := http.Post(server, "text/plain",
				strings.NewReader("defense="+name+":"+part))
			if err == nil {
				resp.Body.Close()
			}
		} else {
			// Обычное сообщение в чат
			full_msg := "[" + name + "]: " + text
			http.Post(server, "text/plain", strings.NewReader(full_msg))
		}
	}
}