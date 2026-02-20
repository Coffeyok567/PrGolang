package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ========== Структуры данных ==========
type Player struct {
	Name    string `json:"name"`
	Attack  string `json:"attack"`
	Defense string `json:"defense"`
	HP      int    `json:"hp"`
}

type ChatMessage struct {
	Sender    string `json:"sender"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type GameState struct {
	Players     map[string]*Player `json:"players"`
	Phase       string              `json:"phase"` // WAIT, ATTACK, DEFENSE, RESULT
	Result      string              `json:"result"`
	PlayerCount int                 `json:"playerCount"`
}

// ========== Глобальные переменные ==========
var (
	// PVP состояние
	players = make(map[string]*Player)
	phase   = "WAIT" // WAIT, ATTACK, DEFENSE, RESULT
	result  string
	pvpMutex sync.RWMutex

	// Чат состояние
	chatHistory []ChatMessage
	chatMutex   sync.RWMutex
)

// Урон по частям тела
var damageByPart = map[string]int{
	"head": 30,
	"body": 20,
	"legs": 10,
}

func main() {
	// Запуск консоли сервера
	go serverConsole()

	// Настройка HTTP маршрутов
	http.HandleFunc("/api/register", handleRegister)
	http.HandleFunc("/api/attack", handleAttack)
	http.HandleFunc("/api/defense", handleDefense)
	http.HandleFunc("/api/chat/send", handleChatSend)
	http.HandleFunc("/api/chat/history", handleChatHistory)
	http.HandleFunc("/api/game/state", handleGameState)
	http.HandleFunc("/api/game/exit", handleExit)

	fmt.Println("🚀 Сервер запущен на http://localhost:8080")
	fmt.Println("📌 Команды сервера: /list, /clear, /help")
	
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
	}
}

// ========== Обработчики HTTP ==========

// Регистрация игрока
func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)
	name := strings.TrimSpace(string(body))

	pvpMutex.Lock()
	defer pvpMutex.Unlock()

	// Проверяем, не полный ли сервер
	if len(players) >= 2 {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "SERVER_FULL",
		})
		return
	}

	// Проверяем, не занято ли имя
	if _, exists := players[name]; exists {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "NAME_TAKEN",
		})
		return
	}

	// Регистрируем игрока
	players[name] = &Player{
		Name:   name,
		HP:     100,
		Attack: "",
		Defense: "",
	}

	fmt.Printf("✅ Игрок зарегистрирован: %s (всего игроков: %d)\n", name, len(players))

	// Отправляем сообщение в чат
	chatMutex.Lock()
	chatHistory = append(chatHistory, ChatMessage{
		Sender:    "СИСТЕМА",
		Message:   fmt.Sprintf("⚔️ Игрок %s присоединился к игре", name),
		Timestamp: time.Now().Unix(),
	})
	chatMutex.Unlock()

	// Если набралось 2 игрока, начинаем игру
	if len(players) == 2 {
		phase = "ATTACK"
		fmt.Println("🎮 ИГРА НАЧАЛАСЬ! Фаза АТАКИ")
		
		chatMutex.Lock()
		chatHistory = append(chatHistory, ChatMessage{
			Sender:    "СИСТЕМА",
			Message:   "⚔️ ИГРА НАЧАЛАСЬ! Фаза АТАКИ (head/body/legs)",
			Timestamp: time.Now().Unix(),
		})
		chatMutex.Unlock()
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "REGISTERED",
	})
}

// Обработка атаки
func handleAttack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Name  string `json:"name"`
		Attack string `json:"attack"`
	}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	pvpMutex.Lock()
	defer pvpMutex.Unlock()

	if phase != "ATTACK" {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "WRONG_PHASE",
		})
		return
	}

	player, exists := players[data.Name]
	if !exists {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "PLAYER_NOT_FOUND",
		})
		return
	}

	player.Attack = data.Attack
	fmt.Printf("⚔️ %s выбрал атаку: %s\n", data.Name, data.Attack)

	// Проверяем, все ли сделали атаку
	if allAttacks() {
		phase = "DEFENSE"
		fmt.Println("🛡️ Фаза ЗАЩИТЫ")
		
		chatMutex.Lock()
		chatHistory = append(chatHistory, ChatMessage{
			Sender:    "СИСТЕМА",
			Message:   "🛡️ Фаза ЗАЩИТЫ (head/body/legs)",
			Timestamp: time.Now().Unix(),
		})
		chatMutex.Unlock()
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "OK",
	})
}

// Обработка защиты
func handleDefense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Name    string `json:"name"`
		Defense string `json:"defense"`
	}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	pvpMutex.Lock()
	defer pvpMutex.Unlock()

	if phase != "DEFENSE" {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "WRONG_PHASE",
		})
		return
	}

	player, exists := players[data.Name]
	if !exists {
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "PLAYER_NOT_FOUND",
		})
		return
	}

	player.Defense = data.Defense
	fmt.Printf("🛡️ %s выбрал защиту: %s\n", data.Name, data.Defense)

	// Проверяем, все ли сделали защиту
	if allDefenses() {
		calculateRound()
		phase = "RESULT"
		
		// Через 8 секунд начинаем новый раунд
		go func() {
			time.Sleep(8 * time.Second)
			pvpMutex.Lock()
			defer pvpMutex.Unlock()
			
			if phase == "RESULT" && checkGameActive() {
				resetRound()
				phase = "ATTACK"
				fmt.Println("⚔️ НОВЫЙ РАУНД! Фаза АТАКИ")
				
				chatMutex.Lock()
				chatHistory = append(chatHistory, ChatMessage{
					Sender:    "СИСТЕМА",
					Message:   "⚔️ НОВЫЙ РАУНД! Фаза АТАКИ",
					Timestamp: time.Now().Unix(),
				})
				chatMutex.Unlock()
			}
		}()
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "OK",
	})
}

// Отправка сообщения в чат
func handleChatSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	chatMutex.Lock()
	chatHistory = append(chatHistory, ChatMessage{
		Sender:    data.Name,
		Message:   data.Message,
		Timestamp: time.Now().Unix(),
	})
	chatMutex.Unlock()

	fmt.Printf("💬 %s: %s\n", data.Name, data.Message)

	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// Получение истории чата
func handleChatHistory(w http.ResponseWriter, r *http.Request) {
	chatMutex.RLock()
	defer chatMutex.RUnlock()
	
	json.NewEncoder(w).Encode(chatHistory)
}

// Получение состояния игры
func handleGameState(w http.ResponseWriter, r *http.Request) {
	pvpMutex.RLock()
	defer pvpMutex.RUnlock()
	
	// Создаем копию players для безопасной отправки
	playersCopy := make(map[string]*Player)
	for k, v := range players {
		playersCopy[k] = &Player{
			Name:    v.Name,
			Attack:  v.Attack,
			Defense: v.Defense,
			HP:      v.HP,
		}
	}
	
	state := GameState{
		Players:     playersCopy,
		Phase:       phase,
		Result:      result,
		PlayerCount: len(players),
	}
	
	json.NewEncoder(w).Encode(state)
}

// Выход игрока
func handleExit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)
	name := strings.TrimSpace(string(body))

	pvpMutex.Lock()
	defer pvpMutex.Unlock()

	delete(players, name)
	fmt.Printf("👋 Игрок вышел: %s\n", name)

	chatMutex.Lock()
	chatHistory = append(chatHistory, ChatMessage{
		Sender:    "СИСТЕМА",
		Message:   fmt.Sprintf("👋 Игрок %s покинул игру", name),
		Timestamp: time.Now().Unix(),
	})
	chatMutex.Unlock()

	if len(players) < 2 {
		phase = "WAIT"
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// ========== Игровая логика ==========

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

func calculateRound() {
	var p1, p2 *Player
	for _, p := range players {
		if p1 == nil {
			p1 = p
		} else {
			p2 = p
		}
	}

	result = "\n=== РЕЗУЛЬТАТ РАУНДА ===\n"
	
	chatMutex.Lock()
	defer chatMutex.Unlock()

	// Атака p1
	if p1.Attack != p2.Defense {
		dmg := damageByPart[p1.Attack]
		p2.HP -= dmg
		if p2.HP < 0 {
			p2.HP = 0
		}
		msg := fmt.Sprintf("⚔️ %s ударил %s в %s (-%d HP)", p1.Name, p2.Name, p1.Attack, dmg)
		result += msg + "\n"
		chatHistory = append(chatHistory, ChatMessage{
			Sender:    "СИСТЕМА",
			Message:   msg,
			Timestamp: time.Now().Unix(),
		})
	} else {
		msg := fmt.Sprintf("🛡️ %s защитился от удара %s", p2.Name, p1.Name)
		result += msg + "\n"
		chatHistory = append(chatHistory, ChatMessage{
			Sender:    "СИСТЕМА",
			Message:   msg,
			Timestamp: time.Now().Unix(),
		})
	}

	// Атака p2
	if p2.Attack != p1.Defense {
		dmg := damageByPart[p2.Attack]
		p1.HP -= dmg
		if p1.HP < 0 {
			p1.HP = 0
		}
		msg := fmt.Sprintf("⚔️ %s ударил %s в %s (-%d HP)", p2.Name, p1.Name, p2.Attack, dmg)
		result += msg + "\n"
		chatHistory = append(chatHistory, ChatMessage{
			Sender:    "СИСТЕМА",
			Message:   msg,
			Timestamp: time.Now().Unix(),
		})
	} else {
		msg := fmt.Sprintf("🛡️ %s защитился от удара %s", p1.Name, p2.Name)
		result += msg + "\n"
		chatHistory = append(chatHistory, ChatMessage{
			Sender:    "СИСТЕМА",
			Message:   msg,
			Timestamp: time.Now().Unix(),
		})
	}

	// Добавляем HP
	hpMsg := fmt.Sprintf("❤️ %s: %d HP | %s: %d HP", p1.Name, p1.HP, p2.Name, p2.HP)
	result += "\n" + hpMsg + "\n"
	chatHistory = append(chatHistory, ChatMessage{
		Sender:    "СИСТЕМА",
		Message:   hpMsg,
		Timestamp: time.Now().Unix(),
	})

	// Проверка на смерть
	if p1.HP <= 0 && p2.HP <= 0 {
		chatHistory = append(chatHistory, ChatMessage{
			Sender:    "СИСТЕМА",
			Message:   "💀 НИЧЬЯ! Оба игрока погибли!",
			Timestamp: time.Now().Unix(),
		})
	} else if p1.HP <= 0 {
		chatHistory = append(chatHistory, ChatMessage{
			Sender:    "СИСТЕМА",
			Message:   fmt.Sprintf("🏆 %s ПОБЕДИЛ! %s повержен!", p2.Name, p1.Name),
			Timestamp: time.Now().Unix(),
		})
	} else if p2.HP <= 0 {
		chatHistory = append(chatHistory, ChatMessage{
			Sender:    "СИСТЕМА",
			Message:   fmt.Sprintf("🏆 %s ПОБЕДИЛ! %s повержен!", p1.Name, p2.Name),
			Timestamp: time.Now().Unix(),
		})
	}

	fmt.Println("📊 Раунд завершен")
}

func resetRound() {
	for _, p := range players {
		p.Attack = ""
		p.Defense = ""
	}
}

func checkGameActive() bool {
	aliveCount := 0
	for _, p := range players {
		if p.HP > 0 {
			aliveCount++
		}
	}
	return aliveCount >= 2
}

// ========== Консоль сервера ==========

func serverConsole() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := scanner.Text()
		
		switch cmd {
		case "/list":
			pvpMutex.RLock()
			fmt.Println("\n=== Игроки ===")
			for name, p := range players {
				fmt.Printf("%s: HP=%d, Attack=%s, Defense=%s\n", 
					name, p.HP, p.Attack, p.Defense)
			}
			fmt.Printf("Фаза: %s\n", phase)
			pvpMutex.RUnlock()
			
		case "/clear":
			chatMutex.Lock()
			chatHistory = []ChatMessage{}
			chatMutex.Unlock()
			fmt.Println("🗑️ Чат очищен")
			
		case "/help":
			fmt.Println("\n=== Команды сервера ===")
			fmt.Println("/list - список игроков")
			fmt.Println("/clear - очистить чат")
			fmt.Println("/help - это меню")
			
		default:
			// Отправляем сообщение от сервера в чат
			if strings.HasPrefix(cmd, "/") {
				fmt.Println("Неизвестная команда")
			} else {
				chatMutex.Lock()
				chatHistory = append(chatHistory, ChatMessage{
					Sender:    "СЕРВЕР",
					Message:   cmd,
					Timestamp: time.Now().Unix(),
				})
				chatMutex.Unlock()
				fmt.Printf("📢 Сервер: %s\n", cmd)
			}
		}
	}
}
