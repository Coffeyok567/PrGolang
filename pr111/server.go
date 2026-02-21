package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ============ ЧАТ ============
var chat_history []string
var history_mutex sync.Mutex
var server_output = make(chan string, 10)

// ============ PVP ИГРА ============
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
	game_mutex sync.Mutex
)

var damageByPart = map[string]int{
	"head": 30,
	"body": 20,
	"legs": 10,
}

// ============ HTML ШАБЛОН ============
const htmlTemplate = `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PvP Чат</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .container {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
            background: rgba(255,255,255,0.1);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 20px;
            box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.37);
        }
        .game-panel {
            background: rgba(0,0,0,0.3);
            border-radius: 15px;
            padding: 20px;
        }
        .chat-panel {
            background: rgba(0,0,0,0.3);
            border-radius: 15px;
            padding: 20px;
            height: 500px;
            display: flex;
            flex-direction: column;
        }
        .messages {
            flex-grow: 1;
            overflow-y: auto;
            margin-bottom: 10px;
            padding: 10px;
            background: rgba(255,255,255,0.1);
            border-radius: 10px;
            font-family: monospace;
        }
        .message {
            margin: 5px 0;
            padding: 5px;
            border-bottom: 1px solid rgba(255,255,255,0.1);
        }
        .player-card {
            background: rgba(255,255,255,0.2);
            border-radius: 10px;
            padding: 15px;
            margin: 10px 0;
            text-align: center;
        }
        .hp-bar {
            width: 100%;
            height: 20px;
            background: #444;
            border-radius: 10px;
            overflow: hidden;
            margin: 10px 0;
        }
        .hp-fill {
            height: 100%;
            background: linear-gradient(90deg, #00ff87 0%, #60efff 100%);
            transition: width 0.3s ease;
        }
        button {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 16px;
            margin: 5px;
            transition: transform 0.2s;
        }
        button:hover {
            transform: scale(1.05);
        }
        input {
            padding: 10px;
            border: none;
            border-radius: 5px;
            width: 70%;
            font-size: 16px;
        }
        .phase {
            font-size: 24px;
            font-weight: bold;
            text-align: center;
            margin: 10px 0;
            color: #ffd700;
        }
        .result-box {
            background: rgba(255,215,0,0.2);
            border: 2px solid gold;
            border-radius: 10px;
            padding: 15px;
            margin: 10px 0;
            white-space: pre-line;
        }
        .controls {
            display: flex;
            flex-wrap: wrap;
            justify-content: center;
            gap: 10px;
        }
        .name-input {
            text-align: center;
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <h1 style="text-align: center;">⚔️ PvP Чат битва ⚔️</h1>
    
    <div class="container">
        <!-- Левая панель: Игра -->
        <div class="game-panel">
            <div class="phase">{{.Phase}}</div>
            
            {{if .ShowNameInput}}
            <div class="name-input">
                <h3>Введите ваше имя для участия в PvP:</h3>
                <input type="text" id="playerName" placeholder="Ваше имя">
                <button onclick="registerForPvP()">Присоединиться к битве</button>
            </div>
            {{end}}
            
            <div class="player-card">
                <h3>Игроки в битве: {{.PlayersCount}}/2</h3>
                {{range .Players}}
                <div style="margin: 10px 0;">
                    <strong>{{.Name}}</strong>
                    <div class="hp-bar">
                        <div class="hp-fill" style="width: {{.HP}}%"></div>
                    </div>
                    <div>❤️ {{.HP}} HP</div>
                </div>
                {{end}}
            </div>
            
            {{if eq .Phase "ATTACK"}}
            <div class="controls">
                <h3>Выберите атаку:</h3>
                <button onclick="sendAttack('head')">👊 Голова (30 урона)</button>
                <button onclick="sendAttack('body')">👊 Тело (20 урона)</button>
                <button onclick="sendAttack('legs')">👊 Ноги (10 урона)</button>
            </div>
            {{end}}
            
            {{if eq .Phase "DEFENSE"}}
            <div class="controls">
                <h3>Выберите защиту:</h3>
                <button onclick="sendDefense('head')">🛡️ Защитить голову</button>
                <button onclick="sendDefense('body')">🛡️ Защитить тело</button>
                <button onclick="sendDefense('legs')">🛡️ Защитить ноги</button>
            </div>
            {{end}}
            
            {{if .Result}}
            <div class="result-box">
                {{.Result}}
            </div>
            {{end}}
        </div>
        
        <!-- Правая панель: Чат -->
        <div class="chat-panel">
            <h2 style="text-align: center;">💬 Чат</h2>
            <div class="messages" id="chatMessages">
                {{range .ChatHistory}}
                <div class="message">{{.}}</div>
                {{end}}
            </div>
            <div style="display: flex; gap: 10px;">
                <input type="text" id="chatInput" placeholder="Введите сообщение..." onkeypress="if(event.key==='Enter') sendMessage()">
                <button onclick="sendMessage()">Отправить</button>
            </div>
            <div style="margin-top: 10px; font-size: 12px; text-align: center;">
                Ваш ник: <span id="currentNick">Гость</span>
            </div>
        </div>
    </div>
    
    <script>
        let playerName = localStorage.getItem('playerName') || 'Гость';
        let pvpName = '';
        document.getElementById('currentNick').textContent = playerName;
        
        function registerForPvP() {
            const name = document.getElementById('playerName').value;
            if (name) {
                pvpName = name;
                fetch('/', {
                    method: 'POST',
                    body: 'register=' + name
                }).then(response => response.text()).then(data => {
                    if (data === 'REGISTERED') {
                        alert('Вы зарегистрированы в PvP режиме!');
                        location.reload();
                    } else if (data === 'SERVER_FULL') {
                        alert('Сервер полон (максимум 2 игрока)');
                    }
                });
            }
        }
        
        function sendAttack(part) {
            if (!pvpName) {
                alert('Сначала зарегистрируйтесь для PvP');
                return;
            }
            fetch('/', {
                method: 'POST',
                body: 'attack=' + pvpName + ':' + part
            });
        }
        
        function sendDefense(part) {
            if (!pvpName) {
                alert('Сначала зарегистрируйтесь для PvP');
                return;
            }
            fetch('/', {
                method: 'POST',
                body: 'defense=' + pvpName + ':' + part
            });
        }
        
        function sendMessage() {
            const input = document.getElementById('chatInput');
            const msg = input.value;
            if (msg) {
                fetch('/', {
                    method: 'POST',
                    body: '[' + playerName + ']: ' + msg
                });
                input.value = '';
            }
        }
        
        function setPlayerName() {
            const newName = prompt('Введите ваш ник в чате:', playerName);
            if (newName) {
                playerName = newName;
                localStorage.setItem('playerName', playerName);
                document.getElementById('currentNick').textContent = playerName;
            }
        }
        
        // Автообновление чата
        let lastMsgCount = 0;
        setInterval(() => {
            fetch('/updates')
                .then(response => response.json())
                .then(data => {
                    const messagesDiv = document.getElementById('chatMessages');
                    messagesDiv.innerHTML = '';
                    data.ChatHistory.forEach(msg => {
                        const div = document.createElement('div');
                        div.className = 'message';
                        div.textContent = msg;
                        messagesDiv.appendChild(div);
                    });
                    messagesDiv.scrollTop = messagesDiv.scrollHeight;
                });
        }, 2000);
        
        // Кнопка смены ника
        document.addEventListener('keydown', function(e) {
            if (e.ctrlKey && e.key === 'n') {
                setPlayerName();
            }
        });
    </script>
</body>
</html>
`

// ============ СТРУКТУРА ДЛЯ ШАБЛОНА ============
type PageData struct {
	Phase        string
	Players      []*Player
	PlayersCount int
	ChatHistory  []string
	Result       string
	ShowNameInput bool
}

// ============ ОСНОВНОЙ ОБРАБОТЧИК ============
func mainHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(r.Body)
		msg := string(body)
		
		// Обработка игровых и чат сообщений
		if strings.HasPrefix(msg, "register=") || 
		   strings.HasPrefix(msg, "attack=") || 
		   strings.HasPrefix(msg, "defense=") {
			handleGameMessage(w, msg)
		} else {
			handleChatMessage(w, msg, getClientIP(r))
		}
	} else {
		// GET запрос - отдаем HTML страницу
		showGamePage(w)
	}
}

// ============ ПОЛУЧЕНИЕ IP КЛИЕНТА ============
func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = strings.Split(r.RemoteAddr, ":")[0]
	}
	return ip
}

// ============ ОТОБРАЖЕНИЕ СТРАНИЦЫ ============
func showGamePage(w http.ResponseWriter) {
	history_mutex.Lock()
	chatCopy := make([]string, len(chat_history))
	copy(chatCopy, chat_history)
	history_mutex.Unlock()
	
	game_mutex.Lock()
	var playersList []*Player
	for _, p := range players {
		playersList = append(playersList, p)
	}
	playersCount := len(players)
	currentPhase := phase
	currentResult := result
	game_mutex.Unlock()
	
	data := PageData{
		Phase:        getPhaseEmoji(currentPhase) + " " + currentPhase,
		Players:      playersList,
		PlayersCount: playersCount,
		ChatHistory:  chatCopy,
		Result:       currentResult,
		ShowNameInput: playersCount < 2,
	}
	
	tmpl := template.New("index")
	tmpl.Parse(htmlTemplate)
	tmpl.Execute(w, data)
}

func getPhaseEmoji(phase string) string {
	switch phase {
	case "WAIT":
		return "⏳"
	case "ATTACK":
		return "⚔️"
	case "DEFENSE":
		return "🛡️"
	case "RESULT":
		return "📊"
	default:
		return ""
	}
}

// ============ ОБНОВЛЕНИЯ ДЛЯ ЧАТА ============
func updatesHandler(w http.ResponseWriter, r *http.Request) {
	history_mutex.Lock()
	defer history_mutex.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ChatHistory": %q}`, chat_history)
}

// ============ ОБРАБОТКА ИГРОВЫХ СООБЩЕНИЙ ============
func handleGameMessage(w http.ResponseWriter, msg string) {
	game_mutex.Lock()
	defer game_mutex.Unlock()

	// РЕГИСТРАЦИЯ
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
		
		addToChat("⚔️ Игрок " + name + " присоединился к битве!")
		
		if len(players) == 2 {
			phase = "ATTACK"
			addToChat("⚔️ БИТВА НАЧИНАЕТСЯ! Игроки выбирают атаку...")
		}
		
		fmt.Fprint(w, "REGISTERED")
		return
	}

	// АТАКА
	if strings.HasPrefix(msg, "attack=") {
		if phase != "ATTACK" {
			fmt.Fprint(w, "WAIT")
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
		
		fmt.Fprint(w, "OK")
		return
	}

	// ЗАЩИТА
	if strings.HasPrefix(msg, "defense=") {
		if phase != "DEFENSE" {
			fmt.Fprint(w, "WAIT")
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
		
		fmt.Fprint(w, "OK")
		return
	}
}

// ============ ОБРАБОТКА ЧАТ-СООБЩЕНИЙ ============
func handleChatMessage(w http.ResponseWriter, msg string, ip string) {
	addToChat(msg)
	server_output <- "💬 " + msg
	fmt.Fprint(w, "получено")
}

func addToChat(msg string) {
	history_mutex.Lock()
	chat_history = append(chat_history, msg)
	if len(chat_history) > 100 { // Храним только последние 100 сообщений
		chat_history = chat_history[1:]
	}
	history_mutex.Unlock()
}

// ============ ИГРОВЫЕ ФУНКЦИИ ============
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
	
	result = "\n⚔️ === РЕЗУЛЬТАТ РАУНДА === ⚔️\n"
	
	// Атака p1
	if p1.Attack != p2.Defense {
		dmg := damageByPart[p1.Attack]
		p2.HP -= dmg
		if p2.HP < 0 {
			p2.HP = 0
		}
		result += fmt.Sprintf(
			"⚔️ %s ударил %s в %s (-%d HP)\n",
			p1.Name, p2.Name, p1.Attack, dmg,
		)
	} else {
		result += fmt.Sprintf(
			"🛡️ %s защитился от удара %s\n",
			p2.Name, p1.Name,
		)
	}
	
	// Атака p2
	if p2.Attack != p1.Defense {
		dmg := damageByPart[p2.Attack]
		p1.HP -= dmg
		if p1.HP < 0 {
			p1.HP = 0
		}
		result += fmt.Sprintf(
			"⚔️ %s ударил %s в %s (-%d HP)\n",
			p2.Name, p1.Name, p2.Attack, dmg,
		)
	} else {
		result += fmt.Sprintf(
			"🛡️ %s защитился от удара %s\n",
			p1.Name, p2.Name,
		)
	}
	
	result += fmt.Sprintf(
		"\n❤️ Здоровье:\n%s = %d HP\n%s = %d HP\n",
		p1.Name, p1.HP,
		p2.Name, p2.HP,
	)
	
	// Проверка на победу
	if p1.HP <= 0 || p2.HP <= 0 {
		winner := p1.Name
		loser := p2.Name
		if p1.HP <= 0 {
			winner = p2.Name
			loser = p1.Name
		}
		result += fmt.Sprintf("\n🏆 %s ПОБЕДИЛ! %s повержен! 🏆\n", winner, loser)
	}
}

func resetRound() {
	for _, p := range players {
		p.Attack = ""
		p.Defense = ""
	}
	phase = "ATTACK"
	addToChat("⚔️ НОВЫЙ РАУНД! Выбирайте атаку...")
}

// ============ КОНСОЛЬНЫЙ ВВОД ============
func consoleInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		
		if strings.HasPrefix(text, "/") {
			handleConsoleCommand(text)
		} else {
			full_msg := "🔴 Сервер: " + text
			addToChat(full_msg)
			server_output <- full_msg
		}
	}
}

func handleConsoleCommand(cmd string) {
	switch cmd {
	case "/reset":
		game_mutex.Lock()
		players = make(map[string]*Player)
		phase = "WAIT"
		result = ""
		game_mutex.Unlock()
		addToChat("🔄 Игра сброшена!")
		server_output <- "🔄 Игра сброшена!"
		
	case "/clear":
		history_mutex.Lock()
		chat_history = []string{}
		history_mutex.Unlock()
		server_output <- "🧹 Чат очищен!"
		
	case "/help":
		server_output <- "📋 Доступные команды:"
		server_output <- "/reset - сброс игры"
		server_output <- "/clear - очистка чата"
		server_output <- "/players - список игроков"
		server_output <- "/help - это меню"
		
	case "/players":
		game_mutex.Lock()
		if len(players) == 0 {
			server_output <- "📋 Нет активных игроков"
		} else {
			server_output <- "📋 Список игроков:"
			for _, p := range players {
				status := "ожидание"
				if p.Attack != "" {
					status = "⚔️ атака выбрана"
				}
				if p.Defense != "" {
					status = "🛡️ защита выбрана"
				}
				server_output <- fmt.Sprintf("%s: ❤️ %d HP (%s)", p.Name, p.HP, status)
			}
		}
		game_mutex.Unlock()
		
	default:
		server_output <- "❌ Неизвестная команда. Введите /help"
	}
}

// ============ MAIN ============
func main() {
	// Канал для вывода сообщений в консоль
	go func() {
		for log_msg := range server_output {
			fmt.Println(log_msg)
		}
	}()
	
	// Горутина для чтения консольного ввода
	go consoleInput()
	
	// Настройка маршрутов
	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/updates", updatesHandler)
	
	// Определяем порт (для Codespace используем 8080)
	port := "8080"
	
	// Получаем URL Codespace
	codespaceName := os.Getenv("CODESPACE_NAME")
	if codespaceName != "" {
		server_output <- fmt.Sprintf("🌐 Ваш публичный URL: https://%s-8080.app.github.dev", codespaceName)
	}
	
	server_output <- fmt.Sprintf("🚀 Сервер запущен на порту %s", port)
	server_output <- "💬 Откройте браузер и перейдите по URL выше"
	server_output <- "📝 В консоли доступны команды: /help"
	
	// Запуск сервера
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		server_output <- "❌ Ошибка запуска сервера: " + err.Error()
	}
}