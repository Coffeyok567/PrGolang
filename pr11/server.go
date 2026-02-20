package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ========== PVP структуры ==========
type Player struct {
	Name    string
	Attack  string
	Defense string
	HP      int
}

// ========== Чат структуры ==========
var chat_history []string
var history_mutex sync.Mutex
var server_output = make(chan string, 10)

// ========== PVP переменные ==========
var (
	players = make(map[string]*Player)
	phase   = "WAIT" // WAIT, ATTACK, DEFENSE, RESULT
	result  string
	pvp_mutex sync.Mutex
)

// Урон по частям тела
var damageByPart = map[string]int{
	"head": 30,
	"body": 20,
	"legs": 10,
}

func main() {
	// Горутина для вывода сообщений из канала в консоль
	go func() {
		for log_msg := range server_output {
			fmt.Println(log_msg)
		}
	}()

	// Горутина для чтения сообщений из консоли сервера
	go func() {
		server_scanner := bufio.NewScanner(os.Stdin)
		for server_scanner.Scan() {
			server_text := server_scanner.Text()
			
			// Добавляем сообщение сервера в историю
			history_mutex.Lock()
			chat_history = append(chat_history, "[СЕРВЕР]: "+server_text)
			history_mutex.Unlock()

			// Отправляем сообщение в канал вывода
			server_output <- "Вы: " + server_text
		}
	}()

	// HTTP-обработчик для всех запросов
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		
		// Обработка POST запросов
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			msg := string(body)

			// Проверяем, является ли запрос чат-сообщением
			if strings.Contains(msg, "[") && strings.Contains(msg, "]:") {
				handleChatMessage(w, msg)
				return
			}

			// Игровые запросы
			pvp_mutex.Lock()
			defer pvp_mutex.Unlock()

			// ===== РЕГИСТРАЦИЯ =====
			if strings.HasPrefix(msg, "register=") {
				name := strings.Split(msg, "=")[1]

				if len(players) >= 2 {
					fmt.Fprint(w, "SERVER_FULL")
					return
				}

				players[name] = &Player{
					Name: name,
					HP:   100,
				}

				// Уведомление в чат о подключении игрока
				history_mutex.Lock()
				chat_history = append(chat_history, fmt.Sprintf("⚔️ [СИСТЕМА]: Игрок %s подключился к PVP", name))
				history_mutex.Unlock()

				if len(players) == 2 {
					phase = "ATTACK"
					history_mutex.Lock()
					chat_history = append(chat_history, "⚔️ [СИСТЕМА]: PVP начался! Оба игрока могут атаковать!")
					history_mutex.Unlock()
				}

				fmt.Fprint(w, "REGISTERED")
				return
			}

			// ===== АТАКА =====
			if strings.HasPrefix(msg, "attack=") {
				if phase != "ATTACK" {
					fmt.Fprint(w, "WAIT")
					return
				}

				parts := strings.Split(strings.Split(msg, "=")[1], ":")
				players[parts[0]].Attack = parts[1]
				
				// Уведомление в чат
				history_mutex.Lock()
				chat_history = append(chat_history, fmt.Sprintf("⚔️ [СИСТЕМА]: %s выбрал атаку", parts[0]))
				history_mutex.Unlock()

				if allAttacks() {
					phase = "DEFENSE"
					history_mutex.Lock()
					chat_history = append(chat_history, "🛡️ [СИСТЕМА]: Фаза защиты! Выберите защиту")
					history_mutex.Unlock()
				}

				fmt.Fprint(w, "OK")
				return
			}

			// ===== ЗАЩИТА =====
			if strings.HasPrefix(msg, "defense=") {
				if phase != "DEFENSE" {
					fmt.Fprint(w, "WAIT")
					return
				}

				parts := strings.Split(strings.Split(msg, "=")[1], ":")
				players[parts[0]].Defense = parts[1]
				
				// Уведомление в чат
				history_mutex.Lock()
				chat_history = append(chat_history, fmt.Sprintf("🛡️ [СИСТЕМА]: %s выбрал защиту", parts[0]))
				history_mutex.Unlock()

				if allDefenses() {
					calcResult()
					phase = "RESULT"
				}

				fmt.Fprint(w, "OK")
				return
			}
		}

		// GET запросы - возвращаем историю чата + игровую информацию
		pvp_mutex.Lock()
		defer pvp_mutex.Unlock()

		// Собираем информацию о PVP
		pvpInfo := ""
		
		if len(players) > 0 {
			pvpInfo += "=== PVP СТАТУС ===\n"
			for name, player := range players {
				status := "⚔️"
				if player.HP <= 0 {
					status = "💀"
				}
				pvpInfo += fmt.Sprintf("%s %s: HP=%d\n", status, name, player.HP)
			}
			pvpInfo += fmt.Sprintf("Фаза: %s\n", phase)
			pvpInfo += "==================\n\n"
		}

		// Отправляем историю чата
		history_mutex.Lock()
		for _, single_msg := range chat_history {
			fmt.Fprintln(w, single_msg)
		}
		history_mutex.Unlock()
		
		// Добавляем PVP информацию
		if pvpInfo != "" {
			fmt.Fprintln(w, pvpInfo)
		}
	})

	server_output <- "🚀 Сервер запущен на :8080"
	server_output <- "📝 Чат и PVP объединены! Игроки могут общаться во время боя"
	
	http.ListenAndServe(":8080", nil)
}

// Обработка чат-сообщений
func handleChatMessage(w http.ResponseWriter, msg string) {
	history_mutex.Lock()
	chat_history = append(chat_history, msg)
	history_mutex.Unlock()
	
	server_output <- "Клиент: " + msg
	fmt.Fprint(w, "получено")
}

// ===== PVP функции =====
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

// ===== РАСЧЁТ УРОНА =====
func calcResult() {
	var p1, p2 *Player
	for _, p := range players {
		if p1 == nil {
			p1 = p
		} else {
			p2 = p
		}
	}

	result = "\n=== РЕЗУЛЬТАТ РАУНДА ===\n"
	
	// Сохраняем результат в историю чата
	history_mutex.Lock()
	defer history_mutex.Unlock()

	// Атака p1
	if p1.Attack != p2.Defense {
		dmg := damageByPart[p1.Attack]
		p2.HP -= dmg
		line := fmt.Sprintf("⚔️ %s ударил %s в %s (-%d HP)", p1.Name, p2.Name, p1.Attack, dmg)
		result += line + "\n"
		chat_history = append(chat_history, "[СИСТЕМА]: "+line)
	} else {
		line := fmt.Sprintf("🛡️ %s защитился от удара %s", p2.Name, p1.Name)
		result += line + "\n"
		chat_history = append(chat_history, "[СИСТЕМА]: "+line)
	}

	// Атака p2
	if p2.Attack != p1.Defense {
		dmg := damageByPart[p2.Attack]
		p1.HP -= dmg
		line := fmt.Sprintf("⚔️ %s ударил %s в %s (-%d HP)", p2.Name, p1.Name, p2.Attack, dmg)
		result += line + "\n"
		chat_history = append(chat_history, "[СИСТЕМА]: "+line)
	} else {
		line := fmt.Sprintf("🛡️ %s защитился от удара %s", p1.Name, p2.Name)
		result += line + "\n"
		chat_history = append(chat_history, "[СИСТЕМА]: "+line)
	}

	// Проверка на смерть
	if p1.HP <= 0 && p2.HP <= 0 {
		chat_history = append(chat_history, "💀 [СИСТЕМА]: Оба игрока погибли! Ничья!")
	} else if p1.HP <= 0 {
		chat_history = append(chat_history, fmt.Sprintf("🏆 [СИСТЕМА]: %s победил! %s повержен!", p2.Name, p1.Name))
	} else if p2.HP <= 0 {
		chat_history = append(chat_history, fmt.Sprintf("🏆 [СИСТЕМА]: %s победил! %s повержен!", p1.Name, p2.Name))
	}

	result += fmt.Sprintf("\nHP:\n%s = %d\n%s = %d\n", p1.Name, p1.HP, p2.Name, p2.HP)
	
	resetRound()
}

func resetRound() {
	for _, p := range players {
		p.Attack = ""
		p.Defense = ""
	}
	phase = "ATTACK"
	
	// Проверяем, не закончилась ли игра
	aliveCount := 0
	for _, p := range players {
		if p.HP > 0 {
			aliveCount++
		}
	}
	
	if aliveCount < 2 {
		phase = "WAIT"
		players = make(map[string]*Player) // Очищаем игроков для новой игры
		chat_history = append(chat_history, "🔄 [СИСТЕМА]: Игра окончена! Можно зарегистрироваться заново")
	}
}
