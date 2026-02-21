package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ... (все предыдущие определения структур и переменных остаются без изменений) ...

// ============ НОВАЯ СТРУКТУРА ДЛЯ ОТВЕТА КЛИЕНТУ ============
type ClientResponse struct {
	ChatHistory []string `json:"chat_history"`
	GameState   GameState `json:"game_state"`
}

type GameState struct {
	Phase        string            `json:"phase"`
	Players      map[string]PlayerInfo `json:"players"`
	PlayersCount int               `json:"players_count"`
	Result       string            `json:"result"`
}

type PlayerInfo struct {
	Name string `json:"name"`
	HP   int    `json:"hp"`
}

// ============ ОСНОВНОЙ ОБРАБОТЧИК ============
func mainHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем User-Agent чтобы понять, кто обращается
	userAgent := r.Header.Get("User-Agent")
	isBrowser := strings.Contains(userAgent, "Mozilla") || 
	             strings.Contains(userAgent, "Chrome") || 
	             strings.Contains(userAgent, "Safari")

	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		msg := string(body)
		
		// Обработка игровых и чат сообщений
		if strings.HasPrefix(msg, "register=") || 
		   strings.HasPrefix(msg, "attack=") || 
		   strings.HasPrefix(msg, "defense=") {
			handleGameMessage(w, msg, isBrowser)
		} else {
			handleChatMessage(w, msg, getClientIP(r), isBrowser)
		}
	} else {
		if isBrowser {
			// Браузеру отдаем HTML страницу
			showGamePage(w)
		} else {
			// Клиенту отдаем JSON с данными
			sendClientData(w)
		}
	}
}

// ============ НОВЫЙ ОБРАБОТЧИК ДЛЯ КЛИЕНТА ============
func sendClientData(w http.ResponseWriter) {
	history_mutex.Lock()
	chatCopy := make([]string, len(chat_history))
	copy(chatCopy, chat_history)
	history_mutex.Unlock()
	
	game_mutex.Lock()
	playersInfo := make(map[string]PlayerInfo)
	for name, p := range players {
		playersInfo[name] = PlayerInfo{
			Name: p.Name,
			HP:   p.HP,
		}
	}
	playersCount := len(players)
	currentPhase := phase
	currentResult := result
	game_mutex.Unlock()
	
	response := ClientResponse{
		ChatHistory: chatCopy,
		GameState: GameState{
			Phase:        currentPhase,
			Players:      playersInfo,
			PlayersCount: playersCount,
			Result:       currentResult,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============ ИСПРАВЛЕННЫЙ ОБРАБОТЧИК ИГРОВЫХ СООБЩЕНИЙ ============
func handleGameMessage(w http.ResponseWriter, msg string, isBrowser bool) {
	game_mutex.Lock()
	defer game_mutex.Unlock()

	// РЕГИСТРАЦИЯ
	if strings.HasPrefix(msg, "register=") {
		name := strings.Split(msg, "=")[1]
		
		if len(players) >= 2 {
			if isBrowser {
				fmt.Fprint(w, "SERVER_FULL")
			} else {
				json.NewEncoder(w).Encode(map[string]string{"status": "SERVER_FULL"})
			}
			return
		}
		
		players[name] = &Player{
			Name: name,
			HP:   100,
		}
		
		addToChat("⚔️ Игрок " + name + " присоединился к битве!")
		
		if len(players) == 2 {
			phase = "ATTACK"
			addToChat("⚔️ БИТВА НАЧИНАЕТСЯ! Игроки выбирают атаку...")
		}
		
		if isBrowser {
			fmt.Fprint(w, "REGISTERED")
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "REGISTERED"})
		}
		return
	}

	// АТАКА
	if strings.HasPrefix(msg, "attack=") {
		if phase != "ATTACK" {
			if isBrowser {
				fmt.Fprint(w, "WAIT")
			} else {
				json.NewEncoder(w).Encode(map[string]string{"status": "WAIT"})
			}
			return
		}
		
		parts := strings.Split(strings.Split(msg, "=")[1], ":")
		if len(parts) == 2 {
			players[parts[0]].Attack = parts[1]
			addToChat("⚔️ " + parts[0] + " готовится к атаке...")
			
			if allAttacks() {
				phase = "DEFENSE"
				addToChat("🛡️ ФАЗА ЗАЩИТЫ! Игроки выбирают защиту...")
			}
		}
		
		if isBrowser {
			fmt.Fprint(w, "OK")
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
		}
		return
	}

	// ЗАЩИТА
	if strings.HasPrefix(msg, "defense=") {
		if phase != "DEFENSE" {
			if isBrowser {
				fmt.Fprint(w, "WAIT")
			} else {
				json.NewEncoder(w).Encode(map[string]string{"status": "WAIT"})
			}
			return
		}
		
		parts := strings.Split(strings.Split(msg, "=")[1], ":")
		if len(parts) == 2 {
			players[parts[0]].Defense = parts[1]
			addToChat("🛡️ " + parts[0] + " принимает защитную стойку...")
			
			if allDefenses() {
				calcResult()
				phase = "RESULT"
				addToChat(result)
				
				// Автоматический переход к следующему раунду через 5 секунд
				go func() {
					time.Sleep(5 * time.Second)
					game_mutex.Lock()
					if phase == "RESULT" {
						resetRound()
					}
					game_mutex.Unlock()
				}()
			}
		}
		
		if isBrowser {
			fmt.Fprint(w, "OK")
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
		}
		return
	}
}

// ============ ИСПРАВЛЕННЫЙ ОБРАБОТЧИК ЧАТА ============
func handleChatMessage(w http.ResponseWriter, msg string, ip string, isBrowser bool) {
	addToChat(msg)
	server_output <- "💬 " + msg
	
	if isBrowser {
		fmt.Fprint(w, "получено")
	} else {
		json.NewEncoder(w).Encode(map[string]string{"status": "received"})
	}
}

// ... (все остальные функции остаются без изменений) ...
