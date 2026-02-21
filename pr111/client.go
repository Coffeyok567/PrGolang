package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type ClientResponse struct {
	ChatHistory []string `json:"chat_history"`
	GameState   struct {
		Phase        string                 `json:"phase"`
		Players      map[string]PlayerInfo  `json:"players"`
		PlayersCount int                    `json:"players_count"`
		Result       string                 `json:"result"`
	} `json:"game_state"`
}

type PlayerInfo struct {
	Name string `json:"name"`
	HP   int    `json:"hp"`
}

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
			var result map[string]string
			json.NewDecoder(resp.Body).Decode(&result)
			if result["status"] == "SERVER_FULL" {
				fmt.Println("❌ Сервер полон (максимум 2 игрока)")
			} else {
				fmt.Println("✅ Вы зарегистрированы в PvP режиме!")
			}
			resp.Body.Close()
		}
	}
	
	// Горутина для вывода сообщений
	go func() {
		for msg := range display_chan {
			fmt.Println(msg)
		}
	}()
	
	// Горутина для получения обновлений
	go func() {
		lastMsgCount := 0
		
		for {
			resp, err := http.Get(server)
			if err == nil {
				var data ClientResponse
				err := json.NewDecoder(resp.Body).Decode(&data)
				if err == nil {
					// Выводим новые сообщения чата
					if len(data.ChatHistory) > lastMsgCount {
						for i := lastMsgCount; i < len(data.ChatHistory); i++ {
							if data.ChatHistory[i] != "" {
								display_chan <- data.ChatHistory[i]
							}
						}
						lastMsgCount = len(data.ChatHistory)
					}
					
					// Выводим состояние игры если изменилось
					if data.GameState.Result != "" {
						display_chan <- "\n" + data.GameState.Result
					}
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
	fmt.Println("💡 Советы: Используйте /status для просмотра состояния игры")
	
	for scanner.Scan() {
		text := scanner.Text()
		
		switch {
		case text == "/status":
			resp, err := http.Get(server)
			if err == nil {
				var data ClientResponse
				json.NewDecoder(resp.Body).Decode(&data)
				fmt.Printf("\n=== СОСТОЯНИЕ ИГРЫ ===\n")
				fmt.Printf("Фаза: %s\n", data.GameState.Phase)
				fmt.Printf("Игроков: %d/2\n", data.GameState.PlayersCount)
				for _, p := range data.GameState.Players {
					fmt.Printf("%s: ❤️ %d HP\n", p.Name, p.HP)
				}
				resp.Body.Close()
			}
			
		case strings.HasPrefix(text, "/attack "):
			part := strings.TrimPrefix(text, "/attack ")
			resp, err := http.Post(server, "text/plain",
				strings.NewReader("attack="+name+":"+part))
			if err == nil {
				resp.Body.Close()
			}
			
		case strings.HasPrefix(text, "/defense "):
			part := strings.TrimPrefix(text, "/defense ")
			resp, err := http.Post(server, "text/plain",
				strings.NewReader("defense="+name+":"+part))
			if err == nil {
				resp.Body.Close()
			}
			
		default:
			// Обычное сообщение в чат
			full_msg := "[" + name + "]: " + text
			http.Post(server, "text/plain", strings.NewReader(full_msg))
		}
	}
}
