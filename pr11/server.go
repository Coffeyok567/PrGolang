package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Структура игрока для PvP режима
type Player struct {
	Name    string
	Attack  string
	Defense string
	HP      int
}

// Структура клиента чата
type ChatClient struct {
	Name string
}

var (
	// PvP данные
	players      = make(map[string]*Player)
	pvpPhase     = "WAIT" // WAIT, ATTACK, DEFENSE, RESULT
	pvpResult    string
	pvpMutex     sync.Mutex

	// Чат данные
	chatHistory  []string
	chatClients  = make(map[string]*ChatClient)
	chatMutex    sync.Mutex

	// Общий мьютекс для всего сервера
	globalMutex  sync.Mutex
)

// Урон по частям тела
var damageByPart = map[string]int{
	"head": 30,
	"body": 20,
	"legs": 10,
}

func main() {
	http.HandleFunc("/", mainHandler)
	
	fmt.Println("Сервер запущен на :8080")
	fmt.Println("Доступные режимы:")
	fmt.Println("- Чат: просто отправляйте сообщения")
	fmt.Println("- PvP: используйте команды register, attack, defense")
	http.ListenAndServe(":8080", nil)
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		msg := string(body)

		// Обработка PvP команд
		if strings.HasPrefix(msg, "register=") {
			handlePvPRegistration(w, msg)
			return
		} else if strings.HasPrefix(msg, "attack=") {
			handleAttack(w, msg)
			return
		} else if strings.HasPrefix(msg, "defense=") {
			handleDefense(w, msg)
			return
		}
		
		// Обработка чат сообщений
		handleChatMessage(w, msg)
		return
	}

	// GET запрос - проверка состояния
	handleGetRequest(w)
}

func handlePvPRegistration(w http.ResponseWriter, msg string) {
	name := strings.Split(msg, "=")[1]

	if len(players) >= 2 {
		fmt.Fprint(w, "SERVER_FULL")
		return
	}

	players[name] = &Player{
		Name: name,
		HP:   100,
	}

	if len(players) == 2 {
		pvpPhase = "ATTACK"
	}

	fmt.Fprint(w, "REGISTERED")
}

func handleAttack(w http.ResponseWriter, msg string) {
	if pvpPhase != "ATTACK" {
		fmt.Fprint(w, "WAIT")
		return
	}

	parts := strings.Split(strings.Split(msg, "=")[1], ":")
	if len(parts) == 2 {
		players[parts[0]].Attack = parts[1]
	}

	if allAttacks() {
		pvpPhase = "DEFENSE"
	}

	fmt.Fprint(w, "OK")
}

func handleDefense(w http.ResponseWriter, msg string) {
	if pvpPhase != "DEFENSE" {
		fmt.Fprint(w, "WAIT")
		return
	}

	parts := strings.Split(strings.Split(msg, "=")[1], ":")
	if len(parts) == 2 {
		players[parts[0]].Defense = parts[1]
	}

	if allDefenses() {
		calcPvPResult()
		pvpPhase = "RESULT"
	}

	fmt.Fprint(w, "OK")
}

func handleChatMessage(w http.ResponseWriter, msg string) {
	// Добавляем сообщение в историю чата
	chatMutex.Lock()
	chatHistory = append(chatHistory, msg)
	chatMutex.Unlock()
	
	// Отправляем подтверждение
	fmt.Fprint(w, "CHAT_MSG_RECEIVED")
}

func handleGetRequest(w http.ResponseWriter) {
	// Сначала проверяем PvP состояние
	if pvpPhase != "WAIT" {
		if pvpPhase == "RESULT" {
			result := pvpResult
			resetPvPRound()
			fmt.Fprint(w, result)
		} else {
			fmt.Fprint(w, pvpPhase)
		}
		return
	}
	
	// Иначе отправляем историю чата
	chatMutex.Lock()
	for _, msg := range chatHistory {
		fmt.Fprintln(w, msg)
	}
	chatMutex.Unlock()
}

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
	if len(players) < 2 {
		return false
	}
	for _, p := range players {
		if p.Defense == "" {
			return false
		}
	}
	return true
}

func calcPvPResult() {
	var p1, p2 *Player
	for _, p := range players {
		if p1 == nil {
			p1 = p
		} else {
			p2 = p
		}
	}

	pvpResult = "=== РЕЗУЛЬТАТ РАУНДА ===\n"

	// Атака p1
	if p1.Attack != p2.Defense {
		dmg := damageByPart[p1.Attack]
		p2.HP -= dmg
		pvpResult += fmt.Sprintf(
			"%s ударил %s в %s (-%d HP)\n",
			p1.Name, p2.Name, p1.Attack, dmg,
		)
	} else {
		pvpResult += fmt.Sprintf(
			"%s защитился от удара %s\n",
			p2.Name, p1.Name,
		)
	}

	// Атака p2
	if p2.Attack != p1.Defense {
		dmg := damageByPart[p2.Attack]
		p1.HP -= dmg
		pvpResult += fmt.Sprintf(
			"%s ударил %s в %s (-%d HP)\n",
			p2.Name, p1.Name, p2.Attack, dmg,
		)
	} else {
		pvpResult += fmt.Sprintf(
			"%s защитился от удара %s\n",
			p1.Name, p2.Name,
		)
	}

	pvpResult += fmt.Sprintf(
		"\nHP:\n%s = %d\n%s = %d\n",
		p1.Name, p1.HP,
		p2.Name, p2.HP,
	)
}

func resetPvPRound() {
	for _, p := range players {
		p.Attack = ""
		p.Defense = ""
	}
	pvpPhase = "ATTACK"
}
