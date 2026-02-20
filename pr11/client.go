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
	Players     map[string]*Player `json:"players"`
	Phase       string              `json:"phase"`
	Result      string              `json:"result"`
	PlayerCount int                 `json:"playerCount"`
}

type Player struct {
	Name    string `json:"name"`
	Attack  string `json:"attack"`
	Defense string `json:"defense"`
	HP      int    `json:"hp"`
}

func main() {
	server := "http://localhost:8080"
	scanner := bufio.NewScanner(os.Stdin)

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
	if !register(server, name) {
		fmt.Println("Нажмите Enter для выхода...")
		scanner.Scan()
		return
	}

	// Запуск горутин для получения данных
	go listenChat(server, name)
	go listenGameState(server, name)

	// Таймер для проверки состояния
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		for range ticker.C {
			checkGameState(server, name)
		}
	}()

	// Основной цикл ввода сообщений
	fmt.Println("\n📝 Просто пишите текст для отправки в чат")
	fmt.Println("💡 Команды: /attack (атака), /defense (защита), /exit - выход")
	fmt.Println("═══════════════════════════════════════════════════════════")
	
	for scanner.Scan() {
		text := scanner.Text()
		
		if text == "/exit" {
			exit(server, name)
			break
		}
		
		if strings.HasPrefix(text, "/attack ") {
			attack := strings.TrimPrefix(text, "/attack ")
			sendAttack(server, name, attack)
		} else if strings.HasPrefix(text, "/defense ") {
			defense := strings.TrimPrefix(text, "/defense ")
			sendDefense(server, name, defense)
		} else if text != "" && !strings.HasPrefix(text, "/") {
			// Отправляем сообщение в чат
			sendMessage(server, name, text)
		}
	}
}

func register(server, name string) bool {
	data := strings.NewReader(name)
	resp, err := http.Post(server+"/api/register", "text/plain", data)
	if err != nil {
		fmt.Println("❌ Ошибка подключения к серверу:", err)
		return false
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
		return false
	}

	fmt.Println("✅ Вы зарегистрированы в PVP!")
	fmt.Println("⏳ Ожидание второго игрока...")
	return true
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

func sendAttack(server, name, attack string) {
	if attack != "head" && attack != "body" && attack != "legs" {
		fmt.Println("❌ Атака должна быть: head, body или legs")
		return
	}
	
	data := map[string]string{
		"name":   name,
		"attack": attack,
	}
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(server+"/api/attack", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		fmt.Println("❌ Ошибка отправки атаки")
		return
	}
	defer resp.Body.Close()
	
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	
	if result["status"] == "success" {
		fmt.Printf("✅ Атака %s отправлена\n", attack)
	} else if result["message"] == "WRONG_PHASE" {
		fmt.Println("❌ Сейчас не фаза атаки")
	}
}

func sendDefense(server, name, defense string) {
	if defense != "head" && defense != "body" && defense != "legs" {
		fmt.Println("❌ Защита должна быть: head, body или legs")
		return
	}
	
	data := map[string]string{
		"name":    name,
		"defense": defense,
	}
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(server+"/api/defense", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		fmt.Println("❌ Ошибка отправки защиты")
		return
	}
	defer resp.Body.Close()
	
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	
	if result["status"] == "success" {
		fmt.Printf("✅ Защита %s отправлена\n", defense)
	} else if result["message"] == "WRONG_PHASE" {
		fmt.Println("❌ Сейчас не фаза защиты")
	}
}

func listenChat(server, name string) {
	lastCount := 0
	for {
		resp, err := http.Get(server + "/api/chat/history")
		if err == nil {
			var messages []ChatMessage
			json.NewDecoder(resp.Body).Decode(&messages)
			resp.Body.Close()

			if len(messages) > lastCount {
				for i := lastCount; i < len(messages); i++ {
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
							fmt.Printf("\033[32m[%s] Вы: %s\033[0m\n", t, msg.Message)
						} else {
							fmt.Printf("\033[37m[%s] %s: %s\033[0m\n", t, msg.Sender, msg.Message)
						}
					}
				}
				lastCount = len(messages)
				fmt.Print("> ")
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

			// Проверяем фазу игры
			if state.PlayerCount == 2 {
				if state.Phase == "ATTACK" {
					if player, exists := state.Players[name]; exists && player.Attack == "" {
						fmt.Printf("\n⚔️ ФАЗА АТАКИ! Используйте: /attack head|body|legs\n> ")
					}
				} else if state.Phase == "DEFENSE" {
					if player, exists := state.Players[name]; exists && player.Defense == "" && player.Attack != "" {
						fmt.Printf("\n🛡️ ФАЗА ЗАЩИТЫ! Используйте: /defense head|body|legs\n> ")
					}
				} else if state.Phase == "RESULT" && state.Result != "" {
					fmt.Printf("\n%s\n> ", state.Result)
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func checkGameState(server, name string) {
	resp, err := http.Get(server + "/api/game/state")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var state GameState
	json.NewDecoder(resp.Body).Decode(&state)

	// Показываем статус игры
	if state.PlayerCount == 2 {
		// Показываем HP игроков
		hpInfo := "❤️ "
		for _, p := range state.Players {
			hpInfo += fmt.Sprintf("%s:%d ", p.Name, p.HP)
		}
		fmt.Printf("\r%s Фаза: %s       ", hpInfo, state.Phase)
	} else if state.PlayerCount == 1 {
		fmt.Printf("\r⏳ Ожидание второго игрока... Всего игроков: %d       ", state.PlayerCount)
	}
}

func checkGameStateSimple(server string) {
	resp, err := http.Get(server + "/api/game/state")
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
