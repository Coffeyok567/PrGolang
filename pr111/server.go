package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// ---------- Игровая логика ----------
type Player struct {
	Name    string
	Attack  string
	Defense string
	HP      int
}

var (
	players = make(map[string]*Player)
	phase   = "WAIT" // WAIT, ATTACK, DEFENSE, RESULT
	result  string
	gameMu  sync.Mutex
)

var damageByPart = map[string]int{
	"head": 30,
	"body": 20,
	"legs": 10,
}

// ---------- Чат ----------
var (
	chatHistory []string
	chatMu      sync.Mutex
)

// ---------- Запуск сервера ----------
func main() {
	// Горутина для ввода сообщений от администратора сервера
	go func() {
		var input string
		for {
			fmt.Scanln(&input)
			if input == "" {
				continue
			}
			fullMsg := "Сервер: " + input
			chatMu.Lock()
			chatHistory = append(chatHistory, fullMsg)
			chatMu.Unlock()
			fmt.Println("Вы:", input)
		}
	}()

	// Основной обработчик (корневой путь) — для чата и игровых команд
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			msg := string(body)

			// Проверяем, является ли сообщение игровой командой
			if strings.HasPrefix(msg, "register=") ||
				strings.HasPrefix(msg, "attack=") ||
				strings.HasPrefix(msg, "defense=") {
				handleGameCommand(w, msg)
				return
			}

			// Обычное сообщение чата
			chatMu.Lock()
			chatHistory = append(chatHistory, msg)
			chatMu.Unlock()
			fmt.Println("Клиент:", msg)
			fmt.Fprint(w, "получено")
			return
		}

		// GET-запрос — возвращаем всю историю чата
		chatMu.Lock()
		defer chatMu.Unlock()
		for _, line := range chatHistory {
			fmt.Fprintln(w, line)
		}
	})

	// Отдельный эндпоинт для получения состояния игры
	http.HandleFunc("/game", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "только GET", http.StatusMethodNotAllowed)
			return
		}
		gameMu.Lock()
		defer gameMu.Unlock()

		switch phase {
		case "WAIT":
			fmt.Fprint(w, "WAIT")
		case "ATTACK":
			fmt.Fprint(w, "ATTACK")
		case "DEFENSE":
			fmt.Fprint(w, "DEFENSE")
		case "RESULT":
			fmt.Fprint(w, result)
			resetRound() // после отправки результата сбрасываем раунд
		default:
			fmt.Fprint(w, "UNKNOWN")
		}
	})

	fmt.Println("Сервер запущен на :8080")
	http.ListenAndServe(":8080", nil)
}

// ---------- Обработка игровых команд ----------
func handleGameCommand(w http.ResponseWriter, cmd string) {
	gameMu.Lock()
	defer gameMu.Unlock()

	switch {
	case strings.HasPrefix(cmd, "register="):
		name := strings.Split(cmd, "=")[1]
		if len(players) >= 2 {
			fmt.Fprint(w, "SERVER_FULL")
			return
		}
		players[name] = &Player{Name: name, HP: 100}
		if len(players) == 2 {
			phase = "ATTACK"
		}
		fmt.Fprint(w, "REGISTERED")

	case strings.HasPrefix(cmd, "attack="):
		if phase != "ATTACK" {
			fmt.Fprint(w, "WAIT")
			return
		}
		parts := strings.Split(strings.Split(cmd, "=")[1], ":")
		if len(parts) != 2 {
			fmt.Fprint(w, "BAD_FORMAT")
			return
		}
		name, part := parts[0], parts[1]
		if _, ok := players[name]; !ok {
			fmt.Fprint(w, "NOT_REGISTERED")
			return
		}
		players[name].Attack = part
		if allAttacks() {
			phase = "DEFENSE"
		}
		fmt.Fprint(w, "OK")

	case strings.HasPrefix(cmd, "defense="):
		if phase != "DEFENSE" {
			fmt.Fprint(w, "WAIT")
			return
		}
		parts := strings.Split(strings.Split(cmd, "=")[1], ":")
		if len(parts) != 2 {
			fmt.Fprint(w, "BAD_FORMAT")
			return
		}
		name, part := parts[0], parts[1]
		if _, ok := players[name]; !ok {
			fmt.Fprint(w, "NOT_REGISTERED")
			return
		}
		players[name].Defense = part
		if allDefenses() {
			calcResult()
			phase = "RESULT"
		}
		fmt.Fprint(w, "OK")
	}
}

// ---------- Вспомогательные функции игры ----------
func allAttacks() bool {
	if len(players) < 2 {
		return false
	}
	for _, p := range players {
		if p.Attack == "" {
			return false
		}
	}
	return true
}

func allDefenses() bool {
	for _, p := range players {
		if p.Defense == "" {
			return false
		}
	}
	return true
}

func calcResult() {
	var p1, p2 *Player
	for _, p := range players {
		if p1 == nil {
			p1 = p
		} else {
			p2 = p
		}
	}

	res := "=== РЕЗУЛЬТАТ РАУНДА ===\n"

	// Атака p1
	if p1.Attack != p2.Defense {
		dmg := damageByPart[p1.Attack]
		p2.HP -= dmg
		res += fmt.Sprintf("%s ударил %s в %s (-%d HP)\n", p1.Name, p2.Name, p1.Attack, dmg)
	} else {
		res += fmt.Sprintf("%s защитился от удара %s\n", p2.Name, p1.Name)
	}

	// Атака p2
	if p2.Attack != p1.Defense {
		dmg := damageByPart[p2.Attack]
		p1.HP -= dmg
		res += fmt.Sprintf("%s ударил %s в %s (-%d HP)\n", p2.Name, p1.Name, p2.Attack, dmg)
	} else {
		res += fmt.Sprintf("%s защитился от удара %s\n", p1.Name, p2.Name)
	}

	res += fmt.Sprintf("\nHP:\n%s = %d\n%s = %d\n", p1.Name, p1.HP, p2.Name, p2.HP)
	result = res
}

func resetRound() {
	for _, p := range players {
		p.Attack = ""
		p.Defense = ""
	}
	phase = "ATTACK"
}
