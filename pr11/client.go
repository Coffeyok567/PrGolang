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

// Структуры данных
type ChatMessage struct {
	Sender    string `json:"sender"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type GameState struct {
	Players map[string]*Player `json:"players"`
	Phase   string              `json:"phase"`
	Result  string              `json:"result"`
}

type Player struct {
	Name    string `json:"name"`
	Attack  string `json:"attack"`
	Defense string `json:"defense"`
	HP      int    `json:"hp"`
	Online  bool   `json:"online"`
}

func main() {
	server := "http://localhost:8080"
	scanner := bufio.NewScanner(os.Stdin)

	// Очищаем экран (работает в большинстве терминалов)
	fmt.Print("\033[H\033[2J")
	
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║     PVP ЧАТ - ИГРА С ОБЩЕНИЕМ        ║")
	fmt.Println("╚══════════════════════════════════════╝")
	
	fmt.Print("\nВведите ваше имя: ")
	scanner.Scan()
	name := strings.TrimSpace(scanner.Text())

	if name == "" {
		fmt.Println("Имя не может быть пустым!")
		return
	}

	// Регистрация в PVP
	register(server, name)

	// Запуск горутин
	go listenGameState(server, name)
	go listenChat(server, name)
	
	// Таймер для обновления интерфейса
	go refreshUI(server, name)

	// Основной цикл ввода сообщений
	fmt.Println("\n📝 Введите сообщение (или '!exit' для выхода):")
	fmt.Println("💡 Для отправки сообщения просто напишите текст")
	
	for scanner.Scan() {
		text := scanner.Text()
		
		if text == "!exit" {
			exit(server, name)
			break
		}
		
		if text != "" {
			// Отправляем сообщение в чат
			sendMessage(server, name, text)
		}
	}
}

func register(server, name string) {
	data := strings.NewReader(name)
	resp, err := http.Post(server+"/api/register", "text/plain", data)
	if err != nil {
		fmt.Println("Ошибка подключения к серверу:", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)

	if result["status"] == "error" {
		switch result["message"] {
		case "SERVER_FULL":
			fmt.Println("❌ Сервер полон! Максимум 2 игрока.")
		case "NAME_TAKEN":
			fmt.Println("❌ Это имя уже занято!")
		default:
			fmt.Println("❌ Ошибка регистрации:", result["message"])
		}
	} else {
		fmt.Println("✅ Вы зарегистрированы в PVP!")
	}
}

func exit(server, name string) {
	http.Post(server+"/api/exit", "text/plain", strings.NewReader(name))
	fmt.Println("👋 До свидания!")
}

func sendMessage(server, name, message string) {
	data := map[string]string{
		"name":    name,
		"message": message,
	}
	jsonData, _ := json.Marshal(data)
	http.Post(server+"/api/chat/send", "application/json", strings.NewReader(string(jsonData)))
}

func listenChat(server, name string) {
	lastIndex := 0
	for {
		resp, err := http.Get(server + "/api/chat/history")
		if err == nil {
			var messages []ChatMessage
			json.NewDecoder(resp.Body).Decode(&messages)
			resp.Body.Close()

			if len(messages) > lastIndex {
				// Сохраняем позицию курсора
				fmt.Print("\033[s")
				
				for i := lastIndex; i < len(messages); i++ {
					msg := messages[i]
					t := time.Unix(msg.Timestamp, 0).Format("15:04:05")
					
					// Разные цвета для разных отправителей
					switch msg.Sender {
					case "СИСТЕМА":
						fmt.Printf("\033[33m[%s] %s: %s\033[0m\n", t, msg.Sender, msg.Message)
					case "СЕРВЕР":
						fmt.Printf("\033[36m[%s] %s: %s\033[0m\n", t, msg.Sender, msg.Message)
					default:
						if msg.Sender == name {
							fmt.Printf("\033[32m[%s] %s: %s\033[0m\n", t, msg.Sender, msg.Message)
						} else {
							fmt.Printf("\033[37m[%s] %s: %s\033[0m\n", t, msg.Sender, msg.Message)
						}
					}
				}
				
				// Возвращаем курсор и показываем приглашение
				fmt.Print("\033[u")
				fmt.Print("📝 Сообщение: ")
				
				lastIndex = len(messages)
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func listenGameState(server, name string) {
	for {
		resp, err := http.Get(server + "/api/game/state")
		if err == nil {
			var state GameState
			json.NewDecoder(resp.Body).Decode(&state)
			resp.Body.Close()

			// Проверяем, наш ли это ход
			if player, exists := state.Players[name]; exists {
				switch state.Phase {
				case "ATTACK":
					if player.Attack == "" {
						// Сохраняем позицию, показываем промпт и возвращаемся
						fmt.Print("\033[s")
						fmt.Print("\n⚔️ Введите атаку (head/body/legs): ")
						fmt.Print("\033[u")
						
						// Читаем ввод в отдельной горутине
						go func() {
							scanner := bufio.NewScanner(os.Stdin)
							if scanner.Scan() {
								attack := strings.TrimSpace(scanner.Text())
								if attack == "head" || attack == "body" || attack == "legs" {
									sendAttack(server, name, attack)
								}
							}
						}()
					}
					
				case "DEFENSE":
					if player.Defense == "" && player.Attack != "" {
						fmt.Print("\033[s")
						fmt.Print("\n🛡️ Введите защиту (head/body/legs): ")
						fmt.Print("\033[u")
						
						go func() {
							scanner := bufio.NewScanner(os.Stdin)
							if scanner.Scan() {
								defense := strings.TrimSpace(scanner.Text())
								if defense == "head" || defense == "body" || defense == "legs" {
									sendDefense(server, name, defense)
								}
							}
						}()
					}
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func refreshUI(server, name string) {
	for {
		resp, err := http.Get(server + "/api/game/state")
		if err == nil {
			var state GameState
			json.NewDecoder(resp.Body).Decode(&state)
			resp.Body.Close()

			// Рисуем верхнюю панель с информацией об игре
			fmt.Print("\033[2J\033[H") // Очищаем экран и ставим курсор в начало
			
			fmt.Println("╔════════════════════════════════════════════════════════════╗")
			fmt.Printf("║  🎮 PVP ЧАТ                         Игрок: %-20s ║\n", name)
			fmt.Println("╠════════════════════════════════════════════════════════════╣")
			
			// Информация об игроках
			players := make([]*Player, 0, 2)
			for _, p := range state.Players {
				players = append(players, p)
			}
			
			if len(players) == 2 {
				p1, p2 := players[0], players[1]
				fmt.Printf("║  %-15s ❤️ %3d HP          %-15s ❤️ %3d HP  ║\n", 
					p1.Name, p1.HP, p2.Name, p2.HP)
			} else if len(players) == 1 {
				fmt.Printf("║  %-15s ❤️ %3d HP          Ожидание игрока...     ║\n", 
					players[0].Name, players[0].HP)
			} else {
				fmt.Println("║  Ожидание игроков...                                  ║")
			}
			
			// Фаза игры
			phaseStr := ""
			switch state.Phase {
			case "WAIT":
				phaseStr = "⏳ Ожидание"
			case "ATTACK":
				phaseStr = "⚔️ АТАКА"
			case "DEFENSE":
				phaseStr = "🛡️ ЗАЩИТА"
			case "RESULT":
				phaseStr = "📊 РЕЗУЛЬТАТ"
			}
			fmt.Printf("║  Фаза: %-20s                               ║\n", phaseStr)
			
			fmt.Println("╠════════════════════════════════════════════════════════════╣")
			fmt.Println("║  ЧАТ СООБЩЕНИЙ:                                           ║")
			fmt.Println("╚════════════════════════════════════════════════════════════╝")
			
			// Возвращаемся к чату
			fmt.Print("\033[10B") // Смещаемся вниз на 10 строк
		}
		time.Sleep(2 * time.Second)
	}
}

func sendAttack(server, name, attack string) {
	data := map[string]string{
		"name":   name,
		"attack": attack,
	}
	jsonData, _ := json.Marshal(data)
	http.Post(server+"/api/attack", "application/json", strings.NewReader(string(jsonData)))
}

func sendDefense(server, name, defense string) {
	data := map[string]string{
		"name":    name,
		"defense": defense,
	}
	jsonData, _ := json.Marshal(data)
	http.Post(server+"/api/defense", "application/json", strings.NewReader(string(jsonData)))
}
