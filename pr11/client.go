package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// ==================== ЧАТ (ФОНОВЫЙ РЕЖИМ) ====================

var chatMessages = make(chan string, 20)
var lastMessageCount = 0
var userName string
var chatRunning = true
var serverAddress = "http://localhost:8080"

// Функция для получения сообщений из чата
func fetchChatMessages() {
	for chatRunning {
		resp, err := http.Get(serverAddress)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			lines := strings.Split(strings.TrimSpace(string(body)), "\n")

			// Если есть новые сообщения
			if len(lines) > lastMessageCount && lines[0] != "" {
				for i := lastMessageCount; i < len(lines); i++ {
					// Не показываем свои сообщения дважды
					if !strings.Contains(lines[i], "["+userName+"]") {
						chatMessages <- lines[i]
					}
				}
				lastMessageCount = len(lines)
			}
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
}

// Функция для отправки сообщения в чат
func sendChatMessage(message string) {
	if message == "" {
		return
	}
	fullMessage := "[" + userName + "]: " + message
	http.Post(serverAddress, "text/plain", strings.NewReader(fullMessage))
}

// Горутина для отображения сообщений чата
func displayChatMessages() {
	for msg := range chatMessages {
		fmt.Printf("\n\033[36m📨 [ЧАТ] %s\033[0m\n", msg)
		fmt.Print("  ➤ ")
	}
}

// ==================== ИГРОВАЯ ЛОГИКА ====================

type BodyPart string

const (
	Head  BodyPart = "голова"
	Torso BodyPart = "торс"
	Legs  BodyPart = "ноги"
)

// Структура "Предмет"
type ItemType string

const (
	WeaponType ItemType = "оружие"
	ArmorType  ItemType = "броня"
	Consumable ItemType = "применяемый предмет"
)

type Item struct {
	Name    string
	Type    ItemType
	Attack  int
	Defence int
	PlusHP  int
}

// Инвентарь
type Equipment struct {
	Weapon *Item
	Armor  *Item
	Gloves *Item
	Helmet *Item
}

type Character interface {
	Hit() BodyPart
	Block() BodyPart
	TakeDamage(damage int)
	IsAlive() bool
	GetName() string
	GetHP() int
	GetStrength() int
}

type BaseCharacter struct {
	Name      string
	HP        int
	MaxHP     int
	Strength  int
	hit       BodyPart
	block     BodyPart
	Inventory []Item // Инвентарь
	Equipment        // Экипированные предметы
}

func (b *BaseCharacter) TakeDamage(damage int) {
	// Учитываем защиту от брони
	totalDefence := 0
	if b.Equipment.Armor != nil {
		totalDefence += b.Equipment.Armor.Defence
	}
	if b.Equipment.Helmet != nil {
		totalDefence += b.Equipment.Helmet.Defence
	}
	if b.Equipment.Gloves != nil {
		totalDefence += b.Equipment.Gloves.Defence
	}

	actualDamage := damage - totalDefence
	if actualDamage < 0 {
		actualDamage = 0
	}

	b.HP -= actualDamage
	if b.HP < 0 {
		b.HP = 0
	}
}

func (b *BaseCharacter) IsAlive() bool {
	return b.HP > 0
}

func (b *BaseCharacter) GetName() string {
	return b.Name
}

func (b *BaseCharacter) GetHP() int {
	return b.HP
}

func (b *BaseCharacter) GetStrength() int {
	// Учитываем атаку от оружия
	totalAttack := b.Strength
	if b.Equipment.Weapon != nil {
		totalAttack += b.Equipment.Weapon.Attack
	}
	return totalAttack
}

func (b *BaseCharacter) SetAttack(target BodyPart) {
	b.hit = target
}

func (b *BaseCharacter) SetBlock(target BodyPart) {
	b.block = target
}

func (b *BaseCharacter) Hit() BodyPart {
	return b.hit
}

func (b *BaseCharacter) Block() BodyPart {
	return b.block
}

type Player struct {
	BaseCharacter
}

// Модифицированная функция выбора с поддержкой чата
func (p *Player) MakeChoice() {
	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Println("                    ВАШ ХОД                   ")
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println("  💬 Чат активен! Просто пишите сообщения")
	fmt.Println("  🔹 Для атаки/защиты вводите цифры")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("  ⚔  Выберите часть тела для АТАКИ:")
		fmt.Println("     1. Голова")
		fmt.Println("     2. Торс")
		fmt.Println("     3. Ноги")
		fmt.Print("  ➤ ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// Проверяем, не сообщение ли это в чат
		if len(input) > 0 && (input[0] < '0' || input[0] > '9') {
			sendChatMessage(input)
			fmt.Println("\n  ✅ Сообщение отправлено в чат!")
			continue
		}

		var attackChoice int
		fmt.Sscanf(input, "%d", &attackChoice)

		if attackChoice >= 1 && attackChoice <= 3 {
			fmt.Println("\n  🛡  Выберите часть тела для ЗАЩИТЫ:")
			fmt.Println("     1. Голова")
			fmt.Println("     2. Торс")
			fmt.Println("     3. Ноги")

			for {
				fmt.Print("  ➤ ")
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				// Проверяем, не сообщение ли это в чат
				if len(input) > 0 && (input[0] < '0' || input[0] > '9') {
					sendChatMessage(input)
					fmt.Println("\n  ✅ Сообщение отправлено в чат!")
					continue
				}

				var defenseChoice int
				fmt.Sscanf(input, "%d", &defenseChoice)

				if defenseChoice >= 1 && defenseChoice <= 3 {
					p.SetAttack(getBodyPart(attackChoice))
					p.SetBlock(getBodyPart(defenseChoice))
					return
				} else {
					fmt.Println("  ⚠ Неверный выбор, попробуйте снова")
				}
			}
		} else {
			fmt.Println("  ⚠ Неверный выбор, попробуйте снова")
		}
	}
}

// Методы для работы с инвентарем
func (p *Player) ShowInventory() {
	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Println("                    ИНВЕНТАРЬ                 ")
	fmt.Println("══════════════════════════════════════════════════")
	if len(p.Inventory) == 0 {
		fmt.Println("\n  📦 Инвентарь пуст")
		return
	}

	fmt.Println()
	for i, item := range p.Inventory {
		fmt.Printf("  %d.", i+1)
		switch item.Type {
		case WeaponType:
			fmt.Printf(" ⚔ %s [АТАКА +%d]", item.Name, item.Attack)
		case ArmorType:
			fmt.Printf(" 🛡 %s [ЗАЩИТА +%d]", item.Name, item.Defence)
		case Consumable:
			fmt.Printf(" 💊 %s [ВОССТ. +%d HP]", item.Name, item.PlusHP)
		}
		fmt.Println()
	}
}

func (p *Player) ShowEquipment() {
	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Println("                  ЭКИПИРОВКА                 ")
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println()
	equipped := false

	if p.Equipment.Weapon != nil {
		fmt.Printf("  ⚔ Оружие:    %s (АТК +%d)\n", p.Equipment.Weapon.Name, p.Equipment.Weapon.Attack)
		equipped = true
	}
	if p.Equipment.Armor != nil {
		fmt.Printf("  🛡 Броня:     %s (ЗАЩ +%d)\n", p.Equipment.Armor.Name, p.Equipment.Armor.Defence)
		equipped = true
	}
	if p.Equipment.Helmet != nil {
		fmt.Printf("  ⛑ Шлем:      %s (ЗАЩ +%d)\n", p.Equipment.Helmet.Name, p.Equipment.Helmet.Defence)
		equipped = true
	}
	if p.Equipment.Gloves != nil {
		fmt.Printf("  ✋ Перчатки:  %s (ЗАЩ +%d)\n", p.Equipment.Gloves.Name, p.Equipment.Gloves.Defence)
		equipped = true
	}

	if !equipped {
		fmt.Println("  Нет надетых предметов")
	}
}

// TakeOff - снять предмет экипировки
func (p *Player) TakeOff() {
	p.ShowEquipment()

	var items []*Item
	var itemNames []string

	if p.Equipment.Weapon != nil {
		items = append(items, p.Equipment.Weapon)
		itemNames = append(itemNames, "Оружие: "+p.Equipment.Weapon.Name)
	}
	if p.Equipment.Armor != nil {
		items = append(items, p.Equipment.Armor)
		itemNames = append(itemNames, "Броня: "+p.Equipment.Armor.Name)
	}
	if p.Equipment.Helmet != nil {
		items = append(items, p.Equipment.Helmet)
		itemNames = append(itemNames, "Шлем: "+p.Equipment.Helmet.Name)
	}
	if p.Equipment.Gloves != nil {
		items = append(items, p.Equipment.Gloves)
		itemNames = append(itemNames, "Перчатки: "+p.Equipment.Gloves.Name)
	}

	if len(items) == 0 {
		fmt.Println("\n  Нечего снимать!")
		return
	}

	fmt.Println("\n  Выберите предмет для снятия:")
	for i, name := range itemNames {
		fmt.Printf("    %d. %s\n", i+1, name)
	}
	fmt.Println("    0. Отмена")
	fmt.Print("  ➤ ")

	var choice int
	fmt.Scan(&choice)

	if choice == 0 || choice > len(items) {
		return
	}

	// Снимаем предмет и добавляем в инвентарь
	itemToRemove := items[choice-1]
	p.Inventory = append(p.Inventory, *itemToRemove)

	// Обнуляем соответствующий слот экипировки
	if p.Equipment.Weapon == itemToRemove {
		p.Equipment.Weapon = nil
	} else if p.Equipment.Armor == itemToRemove {
		p.Equipment.Armor = nil
	} else if p.Equipment.Helmet == itemToRemove {
		p.Equipment.Helmet = nil
	} else if p.Equipment.Gloves == itemToRemove {
		p.Equipment.Gloves = nil
	}

	fmt.Printf("\n  ✨ Снято: %s ✨\n", itemToRemove.Name)
}

// Equip - надеть предмет из инвентаря
func (p *Player) Equip() {
	if len(p.Inventory) == 0 {
		fmt.Println("\n  📦 Инвентарь пуст!")
		return
	}

	p.ShowInventory()
	fmt.Println("\n  Выберите предмет для экипировки:")
	fmt.Println("    0. Отмена")
	fmt.Print("  ➤ ")

	var choice int
	fmt.Scan(&choice)

	if choice == 0 || choice > len(p.Inventory) {
		return
	}

	item := p.Inventory[choice-1]

	// Проверяем, можно ли надеть предмет
	switch item.Type {
	case WeaponType:
		if p.Equipment.Weapon != nil {
			fmt.Printf("\n  ⚠ У вас уже надето оружие: %s\n", p.Equipment.Weapon.Name)
			return
		}
		p.Equipment.Weapon = &item
		fmt.Printf("\n  ⚔ Надето: %s (АТК +%d)\n", item.Name, item.Attack)

	case ArmorType:
		if p.Equipment.Armor != nil {
			fmt.Printf("\n  ⚠ У вас уже надета броня: %s\n", p.Equipment.Armor.Name)
			return
		}
		p.Equipment.Armor = &item
		fmt.Printf("\n  🛡 Надето: %s (ЗАЩ +%d)\n", item.Name, item.Defence)

	default:
		// Для применяемых предметов - используем сразу
		if item.Type == Consumable {
			p.HP += item.PlusHP
			if p.HP > p.MaxHP {
				p.HP = p.MaxHP
			}
			fmt.Printf("\n  💊 Использовано: %s (+%d HP)\n", item.Name, item.PlusHP)
			showHealthBar(p.HP, p.MaxHP, p.Name)
			// Удаляем использованный предмет из инвентаря
			p.Inventory = append(p.Inventory[:choice-1], p.Inventory[choice:]...)
			return
		}
		fmt.Println("\n  ⚠ Этот предмет нельзя надеть")
		return
	}

	// Удаляем предмет из инвентаря
	p.Inventory = append(p.Inventory[:choice-1], p.Inventory[choice:]...)
}

// Функция для создания случайных предметов
func generateRandomItem() Item {
	weapons := []Item{
		{Name: "Меч Грац", Type: WeaponType, Attack: 5},
		{Name: "Ядовитый Кинжал", Type: WeaponType, Attack: 8},
		{Name: "Боевой молот", Type: WeaponType, Attack: 12},
		{Name: "Лук", Type: WeaponType, Attack: 7},
		{Name: "Крысиный посох", Type: WeaponType, Attack: 10},
	}

	armors := []Item{
		{Name: "Кожаная броня", Type: ArmorType, Defence: 3},
		{Name: "Кольчуга", Type: ArmorType, Defence: 6},
		{Name: "Железные доспехи", Type: ArmorType, Defence: 10},
		{Name: "Магическая роба", Type: ArmorType, Defence: 5},
		{Name: "Черепаший панцирь", Type: ArmorType, Defence: 8},
	}

	consumables := []Item{
		{Name: "Малое зелье здоровья", Type: Consumable, PlusHP: 20},
		{Name: "Большое зелье здоровья", Type: Consumable, PlusHP: 50},
		{Name: "Аптечка", Type: Consumable, PlusHP: 30},
		{Name: "Эликсир жизни", Type: Consumable, PlusHP: 80},
		{Name: "Лечебные травы", Type: Consumable, PlusHP: 15},
	}

	allItems := append(append([]Item{}, weapons...), armors...)
	allItems = append(allItems, consumables...)

	return allItems[rand.Intn(len(allItems))]
}

// ==================== PvP (ГОРЯЧИЙ СТУЛ) ====================

type HotSeatBattle struct {
	players       [2]*Player
	round         int
	currentPlayer int // 0 или 1 - индекс текущего игрока
}

func NewHotSeatBattle(player1, player2 *Player) *HotSeatBattle {
	return &HotSeatBattle{
		players:       [2]*Player{player1, player2},
		round:         1,
		currentPlayer: 0,
	}
}

func (h *HotSeatBattle) Start() {
	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Println("           ⚔  PvP - ГОРЯЧИЙ СТУЛ  ⚔             ")
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Printf("\n  👤 Игрок 1: %s\n", h.players[0].GetName())
	showHealthBar(h.players[0].GetHP(), h.players[0].MaxHP, "")
	fmt.Printf("\n  👤 Игрок 2: %s\n", h.players[1].GetName())
	showHealthBar(h.players[1].GetHP(), h.players[1].MaxHP, "")
	fmt.Println()

	for h.players[0].IsAlive() && h.players[1].IsAlive() {
		h.executeRound()
	}

	h.finish()
}

func (h *HotSeatBattle) executeRound() {
	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Printf("                    РАУНД %d                    \n", h.round)
	fmt.Println("══════════════════════════════════════════════════")

	// Ход первого игрока
	fmt.Printf("\n  👤 Ход игрока %s:\n", h.players[0].GetName())
	h.players[0].MakeChoice()

	// Ход второго игрока
	fmt.Printf("\n  👤 Ход игрока %s:\n", h.players[1].GetName())
	h.players[1].MakeChoice()

	h.displayChoices()
	h.processAttacks()
	h.displayStatus()

	h.round++
}

func (h *HotSeatBattle) displayChoices() {
	fmt.Println("\n················································")
	fmt.Println("              ВЫБОРЫ ИГРОКОВ")
	fmt.Println("················································")
	fmt.Printf("\n  👤 %s:\n", h.players[0].GetName())
	fmt.Printf("     ⚔ Атакует:   %s\n", h.players[0].Hit())
	fmt.Printf("     🛡 Защищает:  %s\n", h.players[0].Block())

	fmt.Printf("\n  👤 %s:\n", h.players[1].GetName())
	fmt.Printf("     ⚔ Атакует:   %s\n", h.players[1].Hit())
	fmt.Printf("     🛡 Защищает:  %s\n", h.players[1].Block())
}

func (h *HotSeatBattle) processAttacks() {
	fmt.Println("\n················································")
	fmt.Println("              РЕЗУЛЬТАТЫ АТАК")
	fmt.Println("················································")

	// Атака первого игрока
	player1Damage := h.players[0].GetStrength()
	if h.players[0].Hit() != h.players[1].Block() {
		h.players[1].TakeDamage(player1Damage)
		fmt.Printf("\n  ⚔ %s наносит %d урона %s!\n",
			h.players[0].GetName(), player1Damage, h.players[1].GetName())
	} else {
		fmt.Printf("\n  🛡 %s блокирует удар %s!\n",
			h.players[1].GetName(), h.players[0].GetName())
	}

	// Если второй игрок еще жив, он контратакует
	if h.players[1].IsAlive() && h.players[1].Hit() != h.players[0].Block() {
		player2Damage := h.players[1].GetStrength()
		h.players[0].TakeDamage(player2Damage)
		fmt.Printf("  ⚔ %s наносит %d урона %s!\n",
			h.players[1].GetName(), player2Damage, h.players[0].GetName())
	} else if h.players[1].IsAlive() {
		fmt.Printf("  🛡 %s блокирует удар %s!\n",
			h.players[0].GetName(), h.players[1].GetName())
	}
}

func (h *HotSeatBattle) displayStatus() {
	fmt.Println("\n················································")
	fmt.Println("              ТЕКУЩЕЕ СОСТОЯНИЕ")
	fmt.Println("················································")
	fmt.Printf("\n  👤 %s:\n", h.players[0].GetName())
	showHealthBar(h.players[0].GetHP(), h.players[0].MaxHP, "")
	fmt.Printf("\n  👤 %s:\n", h.players[1].GetName())
	showHealthBar(h.players[1].GetHP(), h.players[1].MaxHP, "")
	fmt.Println()
}

func (h *HotSeatBattle) finish() {
	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Println("                ⚔  PvP ЗАВЕРШЕН  ⚔               ")
	fmt.Println("══════════════════════════════════════════════════")
	if h.players[0].IsAlive() {
		fmt.Printf("\n  🏆 ПОБЕДИТЕЛЬ: %s!\n", h.players[0].GetName())
		fmt.Printf("  💀 ПРОИГРАВШИЙ: %s\n", h.players[1].GetName())
	} else {
		fmt.Printf("\n  🏆 ПОБЕДИТЕЛЬ: %s!\n", h.players[1].GetName())
		fmt.Printf("  💀 ПРОИГРАВШИЙ: %s\n", h.players[0].GetName())
	}
	fmt.Println()
}

// ==================== СЕТЕВОЙ PvP ====================

// Действие игрока
type PlayerAction struct {
	Attack int `json:"attack"`
	Block  int `json:"block"`
}

// Стартовые предметы
func starterItems() []Item {
	return []Item{
		{Name: "Деревянный меч", Type: WeaponType, Attack: 3},
		{Name: "Кожаный жилет", Type: ArmorType, Defence: 2},
		{Name: "Простое зелье здоровья", Type: Consumable, PlusHP: 25},
	}
}

// Сетевой PvP - сервер
func startNetworkServer() {
	fmt.Println("\n╔══════════════════════════════════════════════════╗")
	fmt.Println("║         СЕРВЕР PvP - ОЖИДАНИЕ КЛИЕНТА           ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")

	// Создаем игрока-хоста
	fmt.Print("  Введите ваше имя: ")
	var hostName string
	fmt.Scan(&hostName)

	hostPlayer := &Player{
		BaseCharacter: BaseCharacter{
			Name:      hostName,
			HP:        100,
			MaxHP:     100,
			Strength:  15,
			Inventory: starterItems(),
		},
	}

	// Настройка экипировки перед боем
	fmt.Println("\n  Настройка экипировки перед боем:")
	showInventoryMenu(hostPlayer)

	// Запускаем сервер
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Println("  Ошибка запуска сервера:", err)
		return
	}
	defer listener.Close()

	fmt.Println("\n  Сервер запущен на порту 8081")
	fmt.Println("  Ожидание подключения клиента...")

	// Принимаем соединение
	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("  Ошибка подключения клиента:", err)
		return
	}
	defer conn.Close()

	fmt.Println("  Клиент подключился!")

	// Получаем имя клиента
	clientNameBuf := make([]byte, 1024)
	n, _ := conn.Read(clientNameBuf)
	clientName := string(clientNameBuf[:n])

	// Отправляем имя хоста клиенту
	conn.Write([]byte(hostName))

	fmt.Printf("\n  👤 Хост: %s\n", hostName)
	fmt.Printf("  👤 Клиент: %s\n", clientName)

	// Создаем игрока-клиента
	clientPlayer := &Player{
		BaseCharacter: BaseCharacter{
			Name:      clientName,
			HP:        100,
			MaxHP:     100,
			Strength:  15,
			Inventory: starterItems(),
		},
	}

	// Начинаем бой
	startNetworkBattle(conn, hostPlayer, clientPlayer, true)
}

// Сетевой PvP - клиент
func startNetworkClient() {
	fmt.Println("\n╔══════════════════════════════════════════════════╗")
	fmt.Println("║         КЛИЕНТ PvP - ПОДКЛЮЧЕНИЕ                ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")

	// Вводим адрес сервера
	fmt.Print("  Введите адрес сервера (например, localhost:8081): ")
	var serverAddr string
	fmt.Scan(&serverAddr)

	// Создаем игрока-клиента
	fmt.Print("  Введите ваше имя: ")
	var clientName string
	fmt.Scan(&clientName)

	clientPlayer := &Player{
		BaseCharacter: BaseCharacter{
			Name:      clientName,
			HP:        100,
			MaxHP:     100,
			Strength:  15,
			Inventory: starterItems(),
		},
	}

	// Настройка экипировки перед боем
	fmt.Println("\n  Настройка экипировки перед боем:")
	showInventoryMenu(clientPlayer)

	// Подключаемся к серверу
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("  Ошибка подключения к серверу:", err)
		return
	}
	defer conn.Close()

	fmt.Println("  Подключено к серверу!")

	// Отправляем имя серверу
	conn.Write([]byte(clientName))

	// Получаем имя хоста
	hostNameBuf := make([]byte, 1024)
	n, _ := conn.Read(hostNameBuf)
	hostName := string(hostNameBuf[:n])

	fmt.Printf("\n  👤 Хост: %s\n", hostName)
	fmt.Printf("  👤 Вы: %s\n", clientName)

	// Создаем игрока-хоста
	hostPlayer := &Player{
		BaseCharacter: BaseCharacter{
			Name:      hostName,
			HP:        100,
			MaxHP:     100,
			Strength:  15,
			Inventory: starterItems(),
		},
	}

	// Начинаем бой
	startNetworkBattle(conn, hostPlayer, clientPlayer, false)
}

// Общая логика сетевого боя
func startNetworkBattle(conn net.Conn, hostPlayer, clientPlayer *Player, isHost bool) {
	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	round := 1
	gameOver := false

	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Println("              ⚔  НАЧАЛО PvP БОЯ  ⚔               ")
	fmt.Println("══════════════════════════════════════════════════")

	for !gameOver {
		fmt.Printf("\n══════════════════════════════════════════════════")
		fmt.Printf("\n                    РАУНД %d                    \n", round)
		fmt.Println("══════════════════════════════════════════════════")

		var currentPlayer, otherPlayer *Player

		// Определяем, кто ходит в этом раунде
		if round%2 == 1 { // Нечетные раунды - хост
			if isHost {
				currentPlayer = hostPlayer
				otherPlayer = clientPlayer
				fmt.Println("\n  👤 ВАШ ХОД (вы хост)")
			} else {
				currentPlayer = clientPlayer
				otherPlayer = hostPlayer
				fmt.Println("\n  👤 ХОД СОПЕРНИКА (ожидание)")
			}
		} else { // Четные раунды - клиент
			if !isHost {
				currentPlayer = clientPlayer
				otherPlayer = hostPlayer
				fmt.Println("\n  👤 ВАШ ХОД (вы клиент)")
			} else {
				currentPlayer = hostPlayer
				otherPlayer = clientPlayer
				fmt.Println("\n  👤 ХОД СОПЕРНИКА (ожидание)")
			}
		}

		// Если наш ход
		if (round%2 == 1 && isHost) || (round%2 == 0 && !isHost) {
			// Делаем выбор с поддержкой чата
			fmt.Println("\n  ⚔  Выберите часть тела для АТАКИ:")
			fmt.Println("     1. Голова")
			fmt.Println("     2. Торс")
			fmt.Println("     3. Ноги")
			fmt.Print("  ➤ ")
			
			reader := bufio.NewReader(os.Stdin)
			var attack, block int
			
			// Ввод атаки с поддержкой чата
			for {
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if len(input) > 0 && (input[0] < '0' || input[0] > '9') {
					sendChatMessage(input)
					fmt.Println("\n  ✅ Сообщение отправлено в чат!")
					fmt.Print("  ➤ ")
					continue
				}

				fmt.Sscanf(input, "%d", &attack)
				if attack >= 1 && attack <= 3 {
					break
				} else {
					fmt.Println("  ⚠ Неверный выбор, попробуйте снова")
					fmt.Print("  ➤ ")
				}
			}

			fmt.Println("\n  🛡  Выберите часть тела для ЗАЩИТЫ:")
			fmt.Println("     1. Голова")
			fmt.Println("     2. Торс")
			fmt.Println("     3. Ноги")
			fmt.Print("  ➤ ")

			// Ввод защиты с поддержкой чата
			for {
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if len(input) > 0 && (input[0] < '0' || input[0] > '9') {
					sendChatMessage(input)
					fmt.Println("\n  ✅ Сообщение отправлено в чат!")
					fmt.Print("  ➤ ")
					continue
				}

				fmt.Sscanf(input, "%d", &block)
				if block >= 1 && block <= 3 {
					break
				} else {
					fmt.Println("  ⚠ Неверный выбор, попробуйте снова")
					fmt.Print("  ➤ ")
				}
			}

			currentPlayer.SetAttack(getBodyPart(attack))
			currentPlayer.SetBlock(getBodyPart(block))

			// Отправляем действие сопернику
			action := PlayerAction{Attack: attack, Block: block}
			encoder.Encode(action)

			// Ждем действие соперника
			var otherAction PlayerAction
			decoder.Decode(&otherAction)
			otherPlayer.SetAttack(getBodyPart(otherAction.Attack))
			otherPlayer.SetBlock(getBodyPart(otherAction.Block))

		} else {
			// Ждем действие соперника
			fmt.Println("\n  ⏳ Ожидание хода соперника...")
			fmt.Println("  💬 Вы можете писать в чат:")

			var otherAction PlayerAction
			decoder.Decode(&otherAction)
			otherPlayer.SetAttack(getBodyPart(otherAction.Attack))
			otherPlayer.SetBlock(getBodyPart(otherAction.Block))

			// Делаем свой ход с поддержкой чата
			fmt.Println("\n  ⚔  Выберите часть тела для АТАКИ:")
			fmt.Println("     1. Голова")
			fmt.Println("     2. Торс")
			fmt.Println("     3. Ноги")
			fmt.Print("  ➤ ")
			
			reader := bufio.NewReader(os.Stdin)
			var attack, block int
			
			// Ввод атаки с поддержкой чата
			for {
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if len(input) > 0 && (input[0] < '0' || input[0] > '9') {
					sendChatMessage(input)
					fmt.Println("\n  ✅ Сообщение отправлено в чат!")
					fmt.Print("  ➤ ")
					continue
				}

				fmt.Sscanf(input, "%d", &attack)
				if attack >= 1 && attack <= 3 {
					break
				} else {
					fmt.Println("  ⚠ Неверный выбор, попробуйте снова")
					fmt.Print("  ➤ ")
				}
			}

			fmt.Println("\n  🛡  Выберите часть тела для ЗАЩИТЫ:")
			fmt.Println("     1. Голова")
			fmt.Println("     2. Торс")
			fmt.Println("     3. Ноги")
			fmt.Print("  ➤ ")

			// Ввод защиты с поддержкой чата
			for {
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if len(input) > 0 && (input[0] < '0' || input[0] > '9') {
					sendChatMessage(input)
					fmt.Println("\n  ✅ Сообщение отправлено в чат!")
					fmt.Print("  ➤ ")
					continue
				}

				fmt.Sscanf(input, "%d", &block)
				if block >= 1 && block <= 3 {
					break
				} else {
					fmt.Println("  ⚠ Неверный выбор, попробуйте снова")
					fmt.Print("  ➤ ")
				}
			}

			currentPlayer.SetAttack(getBodyPart(attack))
			currentPlayer.SetBlock(getBodyPart(block))

			// Отправляем действие сопернику
			action := PlayerAction{Attack: attack, Block: block}
			encoder.Encode(action)
		}

		// Показываем выборы
		fmt.Println("\n················································")
		fmt.Println("              РЕЗУЛЬТАТЫ РАУНДА")
		fmt.Println("················································")
		fmt.Printf("\n  👤 %s:\n", hostPlayer.GetName())
		fmt.Printf("     ⚔ Атакует:   %s\n", hostPlayer.Hit())
		fmt.Printf("     🛡 Защищает:  %s\n", hostPlayer.Block())
		fmt.Printf("\n  👤 %s:\n", clientPlayer.GetName())
		fmt.Printf("     ⚔ Атакует:   %s\n", clientPlayer.Hit())
		fmt.Printf("     🛡 Защищает:  %s\n", clientPlayer.Block())

		// Обрабатываем атаки
		fmt.Println("\n················································")
		fmt.Println("              РЕЗУЛЬТАТЫ АТАК")
		fmt.Println("················································")

		// Атака хоста
		if hostPlayer.Hit() != clientPlayer.Block() {
			damage := hostPlayer.GetStrength()
			clientPlayer.TakeDamage(damage)
			fmt.Printf("\n  ⚔ %s наносит %d урона %s!\n",
				hostPlayer.GetName(), damage, clientPlayer.GetName())
		} else {
			fmt.Printf("\n  🛡 %s блокирует удар %s!\n",
				clientPlayer.GetName(), hostPlayer.GetName())
		}

		// Атака клиента (если жив)
		if clientPlayer.IsAlive() && clientPlayer.Hit() != hostPlayer.Block() {
			damage := clientPlayer.GetStrength()
			hostPlayer.TakeDamage(damage)
			fmt.Printf("  ⚔ %s наносит %d урона %s!\n",
				clientPlayer.GetName(), damage, hostPlayer.GetName())
		} else if clientPlayer.IsAlive() {
			fmt.Printf("  🛡 %s блокирует удар %s!\n",
				hostPlayer.GetName(), clientPlayer.GetName())
		}

		// Показываем состояние
		fmt.Println("\n················································")
		fmt.Println("              ТЕКУЩЕЕ СОСТОЯНИЕ")
		fmt.Println("················································")
		fmt.Printf("\n  👤 %s:\n", hostPlayer.GetName())
		showHealthBar(hostPlayer.GetHP(), hostPlayer.MaxHP, "")
		fmt.Printf("\n  👤 %s:\n", clientPlayer.GetName())
		showHealthBar(clientPlayer.GetHP(), clientPlayer.MaxHP, "")

		// Проверка на окончание игры
		if !hostPlayer.IsAlive() || !clientPlayer.IsAlive() {
			gameOver = true
		}

		round++
	}

	// Объявляем победителя
	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Println("                ⚔  БОЙ ЗАВЕРШЕН  ⚔               ")
	fmt.Println("══════════════════════════════════════════════════")
	if hostPlayer.IsAlive() {
		fmt.Printf("\n  🏆 ПОБЕДИТЕЛЬ: %s (хост)!\n", hostPlayer.GetName())
		fmt.Printf("  💀 ПРОИГРАВШИЙ: %s\n", clientPlayer.GetName())
	} else {
		fmt.Printf("\n  🏆 ПОБЕДИТЕЛЬ: %s (клиент)!\n", clientPlayer.GetName())
		fmt.Printf("  💀 ПРОИГРАВШИЙ: %s\n", hostPlayer.GetName())
	}
	fmt.Println()
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ====================

// Вспомогательные функции для красивого интерфейса
func showHealthBar(currentHP, maxHP int, name string) {
	if name != "" {
		fmt.Printf("  %s: ", name)
	} else {
		fmt.Print("     ")
	}

	barWidth := 20
	percent := float64(currentHP) / float64(maxHP)
	filled := int(float64(barWidth) * percent)
	empty := barWidth - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}

	fmt.Printf("[%s] %d/%d ❤\n", bar, currentHP, maxHP)
}

func getBodyPart(choice int) BodyPart {
	switch choice {
	case 1:
		return Head
	case 2:
		return Torso
	case 3:
		return Legs
	default:
		return Torso
	}
}

func displayWelcomeMessage() {
	fmt.Println("\n╔══════════════════════════════════════════════════╗")
	fmt.Println("║         PvP ЧАТ - БИТВА С ОБЩЕНИЕМ              ║")
	fmt.Println("╠══════════════════════════════════════════════════╣")
	fmt.Println("║     💬 Общайтесь с противником во время боя     ║")
	fmt.Println("║     ⚔  PvP режим: Горячий стул                  ║")
	fmt.Println("║     🌐 Сетевой PvP                               ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()
}

func createPlayerForPvP(playerNumber int) *Player {
	var playerName string
	fmt.Printf("  Введите имя игрока %d: ", playerNumber)
	fmt.Scan(&playerName)

	// Даем игроку стартовые предметы
	starterItems := []Item{
		{Name: "Деревянный меч", Type: WeaponType, Attack: 3},
		{Name: "Кожаный жилет", Type: ArmorType, Defence: 2},
		{Name: "Простое зелье здоровья", Type: Consumable, PlusHP: 25},
	}

	return &Player{
		BaseCharacter: BaseCharacter{
			Name:      playerName,
			HP:        100,
			MaxHP:     100,
			Strength:  15,
			Inventory: starterItems,
		},
	}
}

func showInventoryMenu(player *Player) {
	for {
		fmt.Println("\n╔══════════════════════════════════════════════════╗")
		fmt.Println("║                 МЕНЮ ИНВЕНТАРЯ                  ║")
		fmt.Println("╚══════════════════════════════════════════════════╝")
		fmt.Println("  1. Показать инвентарь")
		fmt.Println("  2. Показать экипировку")
		fmt.Println("  3. Надеть предмет")
		fmt.Println("  4. Снять предмет")
		fmt.Println("  5. Начать бой")
		fmt.Print("  ➤ ")

		var choice int
		fmt.Scan(&choice)

		switch choice {
		case 1:
			player.ShowInventory()
		case 2:
			player.ShowEquipment()
		case 3:
			player.Equip()
		case 4:
			player.TakeOff()
		case 5:
			return
		default:
			fmt.Println("\n  ⚠ Неверный выбор")
		}
		fmt.Println()
	}
}

func startPvPMode() {
	fmt.Println("\n╔══════════════════════════════════════════════════╗")
	fmt.Println("║           РЕЖИМ PvP - ГОРЯЧИЙ СТУЛ              ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println("\n  Два игрока будут сражаться на одном компьютере")
	fmt.Println("  💬 Во время боя можно писать в чат\n")

	// Создаем двух игроков
	player1 := createPlayerForPvP(1)
	player2 := createPlayerForPvP(2)

	fmt.Printf("\n  👤 Игрок 1: %s\n", player1.GetName())
	showHealthBar(player1.GetHP(), player1.MaxHP, "")
	fmt.Printf("\n  👤 Игрок 2: %s\n", player2.GetName())
	showHealthBar(player2.GetHP(), player2.MaxHP, "")

	// Даем игрокам возможность настроить экипировку перед боем
	fmt.Println("\n  Настройка экипировки перед боем:")
	fmt.Println("\n  👤 Игрок 1, настройте свою экипировку:")
	showInventoryMenu(player1)

	fmt.Println("\n  👤 Игрок 2, настройте свою экипировку:")
	showInventoryMenu(player2)

	// Начинаем бой PvP
	battle := NewHotSeatBattle(player1, player2)
	battle.Start()
}

func showMainMenu() {
	fmt.Println("\n╔══════════════════════════════════════════════════╗")
	fmt.Println("║                  ГЛАВНОЕ МЕНЮ                    ║")
	fmt.Println("╠══════════════════════════════════════════════════╣")
	fmt.Println("║  1. PvP (Горячий стул) - на одном компьютере    ║")
	fmt.Println("║  2. СЕТЕВОЙ PvP - создать сервер                 ║")
	fmt.Println("║  3. СЕТЕВОЙ PvP - подключиться к серверу         ║")
	fmt.Println("║  4. Выйти из игры                                ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Print("  ➤ ")
}

// ==================== ОСНОВНАЯ ФУНКЦИЯ ====================

func main() {
	rand.Seed(time.Now().UnixNano())

	// Запрашиваем имя для чата
	fmt.Print("\n  Введите ваше имя для чата: ")
	fmt.Scan(&userName)

	// Запускаем фоновый чат
	go fetchChatMessages()
	go displayChatMessages()

	// Отправляем приветственное сообщение
	sendChatMessage("подключился к игре!")

	// Даем чату время на инициализацию
	time.Sleep(1 * time.Second)

	displayWelcomeMessage()

	// Главное меню
	for {
		showMainMenu()

		var choice int
		fmt.Scan(&choice)

		switch choice {
		case 1:
			startPvPMode()
		case 2:
			// Сетевой PvP - сервер
			startNetworkServer()
		case 3:
			// Сетевой PvP - клиент
			startNetworkClient()
		case 4:
			// Отправляем сообщение о выходе
			sendChatMessage("покинул игру")
			chatRunning = false
			time.Sleep(1 * time.Second)

			fmt.Println("\n╔══════════════════════════════════════════════════╗")
			fmt.Println("║                 ДО СВИДАНИЯ!                    ║")
			fmt.Println("╚══════════════════════════════════════════════════╝")
			fmt.Println("\n  Спасибо за игру! Возвращайтесь скорее!")
			return
		default:
			fmt.Println("\n  ⚠ Неверный выбор")
		}

		fmt.Print("\n  Нажмите Enter, чтобы продолжить...")
		fmt.Scanln()
	}
}
