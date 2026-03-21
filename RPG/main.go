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
var serverAddress = "https://improved-guide-v6qv54v796r5cjx5-8080.app.github.dev/"

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

// ==================== ИГРОВАЯ ЛОГИКА (ОСНОВНАЯ) ====================

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

type Player struct {
	BaseCharacter
}

func (p *Player) Hit() BodyPart {
	return p.hit
}

func (p *Player) Block() BodyPart {
	return p.block
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

		// Если пустой ввод, пропускаем
		if input == "" {
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

				// Если пустой ввод, пропускаем
				if input == "" {
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

// Трофеи - добавляем предмет врагу
type Enemy struct {
	BaseCharacter
	Loot Item // Предмет, который выпадает при победе
}

func (e *Enemy) Hit() BodyPart {
	return e.hit
}

func (e *Enemy) Block() BodyPart {
	return e.block
}

func (e *Enemy) MakeRandomChoice() {
	parts := []BodyPart{Head, Torso, Legs}
	e.SetAttack(parts[rand.Intn(len(parts))])
	e.SetBlock(parts[rand.Intn(len(parts))])
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

type Battle struct {
	player *Player
	enemy  *Enemy
	round  int
}

func NewBattle(player *Player, enemy *Enemy) *Battle {
	return &Battle{
		player: player,
		enemy:  enemy,
		round:  1,
	}
}

func (b *Battle) Start() bool {
	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Println("                ⚔  НАЧАЛО БОЯ  ⚔                 ")
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Printf("\n  🐷 %s\n", b.player.GetName())
	showHealthBar(b.player.GetHP(), b.player.MaxHP, "")
	fmt.Printf("\n  👾 %s\n", b.enemy.GetName())
	showHealthBar(b.enemy.GetHP(), b.enemy.MaxHP, "")
	fmt.Println()

	for b.player.IsAlive() && b.enemy.IsAlive() {
		b.executeRound()
	}

	b.finish()

	// Если игрок победил, добавляем трофей в инвентарь
	if b.player.IsAlive() && b.enemy.Loot.Name != "" {
		fmt.Println("\n══════════════════════════════════════════════════")
		fmt.Println("                     🏆 ПОБЕДА 🏆                  ")
		fmt.Println("══════════════════════════════════════════════════")
		fmt.Printf("\n  Вы получили трофей: %s!\n", b.enemy.Loot.Name)
		b.player.Inventory = append(b.player.Inventory, b.enemy.Loot)
	}

	return b.player.IsAlive()
}

func (b *Battle) executeRound() {
	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Printf("                    РАУНД %d                    \n", b.round)
	fmt.Println("══════════════════════════════════════════════════")

	b.player.MakeChoice()
	b.enemy.MakeRandomChoice()

	b.displayChoices()
	b.processAttacks()
	b.displayStatus()

	b.round++
}

func (b *Battle) displayChoices() {
	fmt.Println("\n················································")
	fmt.Println("              ВЫБОРЫ ИГРОКОВ")
	fmt.Println("················································")
	fmt.Printf("\n  🐷 %s:\n", b.player.GetName())
	fmt.Printf("     ⚔ Атакует:   %s\n", b.player.Hit())
	fmt.Printf("     🛡 Защищает:  %s\n", b.player.Block())

	fmt.Printf("\n  👾 %s:\n", b.enemy.GetName())
	fmt.Printf("     ⚔ Атакует:   %s\n", b.enemy.Hit())
	fmt.Printf("     🛡 Защищает:  %s\n", b.enemy.Block())
}

func (b *Battle) processAttacks() {
	fmt.Println("\n················································")
	fmt.Println("              РЕЗУЛЬТАТЫ АТАК")
	fmt.Println("················································")

	playerDamage := b.player.GetStrength()
	if b.player.Hit() != b.enemy.Block() {
		b.enemy.TakeDamage(playerDamage)
		fmt.Printf("\n  ⚔ %s наносит %d урона %s!\n",
			b.player.GetName(), playerDamage, b.enemy.GetName())
	} else {
		fmt.Printf("\n  🛡 %s блокирует удар %s!\n",
			b.enemy.GetName(), b.player.GetName())
	}

	if b.enemy.IsAlive() && b.enemy.Hit() != b.player.Block() {
		enemyDamage := b.enemy.GetStrength()
		b.player.TakeDamage(enemyDamage)
		fmt.Printf("  ⚔ %s наносит %d урона %s!\n",
			b.enemy.GetName(), enemyDamage, b.player.GetName())
	} else if b.enemy.IsAlive() {
		fmt.Printf("  🛡 %s блокирует удар %s!\n",
			b.player.GetName(), b.enemy.GetName())
	}
}

func (b *Battle) displayStatus() {
	fmt.Println("\n················································")
	fmt.Println("              ТЕКУЩЕЕ СОСТОЯНИЕ")
	fmt.Println("················································")
	fmt.Printf("\n  🐷 %s:\n", b.player.GetName())
	showHealthBar(b.player.GetHP(), b.player.MaxHP, "")
	fmt.Printf("\n  👾 %s:\n", b.enemy.GetName())
	showHealthBar(b.enemy.GetHP(), b.enemy.MaxHP, "")
	fmt.Println()
}

func (b *Battle) finish() {
	fmt.Println("\n══════════════════════════════════════════════════")
	fmt.Println("                ⚔  БОЙ ЗАВЕРШЕН  ⚔               ")
	fmt.Println("══════════════════════════════════════════════════")
	if b.player.IsAlive() {
		fmt.Printf("\n  🏆 ПОБЕДИТЕЛЬ: %s!\n", b.player.GetName())
		fmt.Printf("  💀 ПРОИГРАВШИЙ: %s\n", b.enemy.GetName())
	} else {
		fmt.Printf("\n  🏆 ПОБЕДИТЕЛЬ: %s!\n", b.enemy.GetName())
		fmt.Printf("  💀 ПРОИГРАВШИЙ: %s\n", b.player.GetName())
	}
	fmt.Println()
}

// HotSeatBattle - структура для режима "Горячий стул"
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
	fmt.Printf("\n  👤 Ход игрока %s:\n", h.players[h.currentPlayer].GetName())
	h.players[h.currentPlayer].MakeChoice()

	// Ход второго игрока
	otherPlayer := (h.currentPlayer + 1) % 2
	fmt.Printf("\n  👤 Ход игрока %s:\n", h.players[otherPlayer].GetName())
	h.players[otherPlayer].MakeChoice()

	h.displayChoices()
	h.processAttacks()
	h.displayStatus()

	h.round++
	// Переход хода к следующему игроку
	h.currentPlayer = otherPlayer
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

// ==================== ИГРОВАЯ ЛОГИКА (САМУРАЙ) ====================

const (
	SamuraiHead = 1
	SamuraiBody = 2
	SamuraiLegs = 3
)

// SamuraiItem - описание любого предмета в игре (самурайская версия)
type SamuraiItem struct {
	Name    string
	Type    string
	Attack  int
	Defense int
	PlusHP  int
}

// SamuraiPlayer - вся информация об игроке (самурайская версия)
type SamuraiPlayer struct {
	Name      string
	HP        int
	MaxHP     int
	Strength  int
	Inventory []SamuraiItem
	Equipped  map[string]SamuraiItem
}

// SamuraiEnemy - информация о враге (самурайская версия)
type SamuraiEnemy struct {
	Name     string
	HP       int
	Strength int
	Loot     SamuraiItem
}

// Equip - надеть предмет или положить лечебное снадобье в рюкзак
func (p *SamuraiPlayer) Equip(item SamuraiItem) {
	if item.Type == "хилка" {
		p.Inventory = append(p.Inventory, item)
		fmt.Printf(">> В инвентарь добавлены: %s\n", item.Name)
		return
	}
	if _, exists := p.Equipped[item.Type]; exists {
		fmt.Printf("! У вас уже надета %s. Старая заменена на %s.\n", item.Type, item.Name)
	}
	p.Equipped[item.Type] = item
	fmt.Printf(">> %s экипировал: %s\n", p.Name, item.Name)
}

// UseHeal - найти и использовать лечебное снадобье в рюкзаке
func (p *SamuraiPlayer) UseHeal() {
	for i, item := range p.Inventory {
		if item.Type == "хилка" {
			p.HP += item.PlusHP
			if p.HP > p.MaxHP {
				p.HP = p.MaxHP
			}
			fmt.Printf("++ %s использовал целебный отвар (+%d HP). Текущее HP: %d\n", p.Name, item.PlusHP, p.HP)
			p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
			return
		}
	}
	fmt.Println("У вас закончились целебные отвары!")
}

// calculateSamuraiDamage - считает, сколько урона нанесет атакующий
func calculateSamuraiDamage(attackerStr int, attackerEquip map[string]SamuraiItem, defenderEquip map[string]SamuraiItem) int {
	dmg := attackerStr
	// добавляем урон от оружия (если оно есть)
	if val, ok := attackerEquip["оружие"]; ok {
		dmg += val.Attack
	}
	if val, ok := defenderEquip["броня"]; ok {
		dmg -= val.Defense
	}
	if dmg < 5 {
		dmg = 5
	}
	return dmg
}

// samuraiFight - главная функция битвы с врагом (возвращает true, если игрок победил)
func samuraiFight(p *SamuraiPlayer, enemy *SamuraiEnemy) bool {
	fmt.Printf("\n--- БИТВА: %s против %s ---\n", p.Name, enemy.Name)

	for p.HP > 0 && enemy.HP > 0 {
		fmt.Printf("\n[ %s: %d/%d HP ] vs [ %s: %d HP ]\n", p.Name, p.HP, p.MaxHP, enemy.Name, enemy.HP)
		fmt.Println("1. Атака  2. Лечение  3. Осмотреть врага")
		var choice int
		fmt.Scan(&choice)

		if choice == 2 {
			p.UseHeal()
			continue
		} else if choice == 3 {
			fmt.Printf("Враг %s выглядит опасным. Его сила примерно %d.\n", enemy.Name, enemy.Strength)
			continue
		}

		fmt.Println("Выберите зону АТАКИ (1-Голова, 2-Тело, 3-Ноги):")
		var p_hit int
		fmt.Scan(&p_hit)
		fmt.Println("Выберите зону ЗАЩИТЫ (1-Голова, 2-Тело, 3-Ноги):")
		var p_block int
		fmt.Scan(&p_block)

		e_hit := rand.Intn(3) + 1
		e_block := rand.Intn(3) + 1

		// ХОД ИГРОКА: атакуем врага
		if p_hit != e_block {
			// если враг не заблокировал - наносим урон
			dmg := calculateSamuraiDamage(p.Strength, p.Equipped, nil)
			enemy.HP -= dmg
			fmt.Printf("УДАР! Вы нанесли %d урона.\n", dmg)
		} else {
			fmt.Println("Враг ПАРИРОВАЛ ваш удар!")
		}

		// ХОД ВРАГА: если враг еще жив - он атакует
		if enemy.HP > 0 {
			if e_hit != p_block {
				// если игрок не заблокировал - получаем урон
				dmg := calculateSamuraiDamage(enemy.Strength, nil, p.Equipped)
				p.HP -= dmg
				fmt.Printf("КРИТ! %s нанес вам %d урона.\n", enemy.Name, dmg)
			} else {
				fmt.Println("ИДЕАЛЬНОЕ ПАРИРОВАНИЕ! Вы не получили урона.")
			}
		}
	}
	return p.HP > 0
}

func storyMode() {
	fmt.Println("\n==========================================")
	fmt.Println("    ПУТЬ САМУРАЯ: ИСТОРИЯ МЕЧА И ЧЕСТИ   ")
	fmt.Println("==========================================")
	fmt.Println("Япония, XVI век. Эпоха воюющих провинций - Сэнгоку Дзидай.")
	fmt.Println("Вы - Такэда Хидэо, молодой самурай из разоренного клана.")
	fmt.Println("Ваш отец погиб в битве, защищая честь семьи. Месть и долг зовут вас в путь...")

	// создаем игрока для сюжета
	p := SamuraiPlayer{
		Name:      "Такэда Хидэо",
		HP:        100,
		MaxHP:     100,
		Strength:  12,
		Equipped:  make(map[string]SamuraiItem),
		Inventory: []SamuraiItem{},
	}

	// даем стартовый меч
	p.Equip(SamuraiItem{"Меч отца (Катана рода Такэда)", "оружие", 8, 0, 0})
	// даем одно лечебное снадобье для начала
	p.Inventory = append(p.Inventory, SamuraiItem{"Целебный отвар бабушки", "хилка", 0, 0, 30})

	chapters := []struct {
		Title string
		Story string
		Enemy SamuraiEnemy
		After string
	}{
		{ // Глава 1
			Title: "Глава 1: Сожженный храм",
			Story: "Вы приходите к семейному храму, но находите лишь пепелище. Банда ронинов разграбила святыню и осквернила могилы предков.",
			Enemy: SamuraiEnemy{"Предводитель ронинов", 45, 8, SamuraiItem{"Ржавый, но острый меч", "оружие", 6, 0, 0}},
			After: "Вы отомстили за поруганные могилы. В обломках храма вы находите старый свиток с техниками боя.",
		},
		{ // Глава 2
			Title: "Глава 2: Встреча в бамбуковой роще",
			Story: "На горной тропе вам преграждает путь знаменитый воин клана врагов. Он насмехается над памятью вашего отца.",
			Enemy: SamuraiEnemy{"Самурай клана Акаги", 65, 14, SamuraiItem{"Доспехи павшего героя", "броня", 0, 9, 0}},
			After: "Поверженный враг признает вашу силу. Вы забираете его доспехи как трофей. Путь к замку Акаги открыт.",
		},
		{ // Глава 3
			Title: "Глава 3: Тень ниндзя",
			Story: "Слухи о вашем гневе дошли до предводителей. Ночью на вас нападает ниндзя, нанятый кланом Акаги. Тишина и смерть следуют за ним.",
			Enemy: SamuraiEnemy{"Ниндзя-убийца", 55, 16, SamuraiItem{"Яд гадюки", "хилка", 0, 0, 40}},
			After: "Вы отразили ночную атаку. Яд ниндзя может пригодиться - в умелых руках он лечит лучше любых трав.",
		},
		{ // Глава 4 (финальная)
			Title: "Глава 4: Замок Акаги",
			Story: "Вы стоите у ворот вражеского замка. Здесь правит Даймё Акаги Нобутада - человек, который убил вашего отца. Последний бой за честь рода.",
			Enemy: SamuraiEnemy{"Даймё Акаги Нобутада", 100, 20, SamuraiItem{"Клинок рода Акаги", "оружие", 12, 0, 0}},
			After: "Даймё пал от вашей руки. Дух отца наконец обретает покой. Легенда о вашей мести разнесется по всем провинциям.",
		},
	}

	for i, ch := range chapters {
		fmt.Printf("\n=== %s ===\n", ch.Title)
		fmt.Printf("«%s»\n", ch.Story)

		if samuraiFight(&p, &ch.Enemy) {
			fmt.Printf("\nПОБЕДА! %s\n", ch.After)
			p.Equip(ch.Enemy.Loot)

			// даем выбор бонуса за победу
			fmt.Println("\nВаш дух самурая крепнет. Выберите путь развития:")
			fmt.Println("1. Крепкое тело (+20 Макс. HP)")
			fmt.Println("2. Твердая рука (+5 Сила)")
			var up int
			fmt.Scan(&up)
			if up == 1 {
				p.MaxHP += 20
				p.HP += 20
				fmt.Println("Вы стали выносливее!")
			} else {
				p.Strength += 5
				fmt.Println("Ваши удары стали сильнее!")
			}
		} else {
			fmt.Println("\nСмерть приняла вас с честью... Но клан угас, а враги торжествуют.")
			fmt.Println("Ваша история не будет рассказана потомкам.")
			return
		}

		if i < len(chapters)-1 {
			fmt.Println("\nВы разбиваете лагерь у подножия горы. Медитация и отдых восстанавливают силы... (HP восстановлено на 30)")
			p.HP += 30
			if p.HP > p.MaxHP {
				p.HP = p.MaxHP
			}
		}
	}

	fmt.Println("\n══════════════════════════════════════════════════════════")
	fmt.Println("ЭПИЛОГ: Месть свершилась, но сердце самурая не знает покоя.")
	fmt.Println("Вы стоите на балконе замка Акаги, глядя на восходящее солнце.")
	fmt.Println("Враги пали, но честь рода Такэда восстановлена навеки.")
	fmt.Println("Ваше имя вписано в свитки истории золотыми иероглифами.")
	fmt.Println("Ведь вы не просто воин. Вы — САМУРАЙ.")
	fmt.Println("══════════════════════════════════════════════════════════")
}

// --- РЕЖИМ PVP (игрок против игрока) упрощенный ---

func simplePvPMode() {
	// создаем двух игроков
	p1 := SamuraiPlayer{Name: "Самурай Востока", HP: 100, MaxHP: 100, Strength: 15, Equipped: make(map[string]SamuraiItem), Inventory: []SamuraiItem{}}
	p2 := SamuraiPlayer{Name: "Самурай Запада", HP: 100, MaxHP: 100, Strength: 15, Equipped: make(map[string]SamuraiItem), Inventory: []SamuraiItem{}}

	// даем обоим по катане
	katana := SamuraiItem{"Острая Катана", "оружие", 10, 0, 0}
	p1.Equip(katana)
	p2.Equip(katana)

	fmt.Println("\n══════════════════════════════════════════════════════════")
	fmt.Println("        РЕЖИМ PVP: ДУЭЛЬ ДВУХ САМУРАЕВ        ")
	fmt.Println("══════════════════════════════════════════════════════════")
	fmt.Println("Только честный бой решит, кто достоин носить имя воина!")

	// бой идет, пока кто-то не умрет
	for p1.HP > 0 && p2.HP > 0 {
		// показываем состояние
		fmt.Printf("\n⚔️ %s (%d HP)  |  ⚔️ %s (%d HP)\n", p1.Name, p1.HP, p2.Name, p2.HP)

		// первый игрок выбирает атаку и защиту
		fmt.Printf("\nХод %s.\nАтака (1-Голова, 2-Тело, 3-Ноги): ", p1.Name)
		var a1 int
		fmt.Scan(&a1)
		fmt.Printf("Защита (1-Голова, 2-Тело, 3-Ноги): ")
		var d1 int
		fmt.Scan(&d1)

		// второй игрок выбирает атаку и защиту
		fmt.Printf("\nХод %s.\nАтака (1-Голова, 2-Тело, 3-Ноги): ", p2.Name)
		var a2 int
		fmt.Scan(&a2)
		fmt.Printf("Защита (1-Голова, 2-Тело, 3-Ноги): ")
		var d2 int
		fmt.Scan(&d2)

		// применяем урон (если атака не совпала с защитой)
		if a1 != d2 {
			dmg := calculateSamuraiDamage(p1.Strength, p1.Equipped, p2.Equipped)
			p2.HP -= dmg
			fmt.Printf("⚡ %s нанес %d урона!\n", p1.Name, dmg)
		} else {
			fmt.Printf("🛡️ %s парировал удар!\n", p2.Name)
		}

		if a2 != d1 {
			dmg := calculateSamuraiDamage(p2.Strength, p2.Equipped, p1.Equipped)
			p1.HP -= dmg
			fmt.Printf("⚡ %s нанес %d урона!\n", p2.Name, dmg)
		} else {
			fmt.Printf("🛡️ %s парировал удар!\n", p1.Name)
		}
	}

	// объявляем победителя
	fmt.Println("\n══════════════════════════════════════════════════════════")
	if p1.HP > 0 {
		fmt.Printf("🏆 ПОБЕДИТЕЛЬ: %s! Честь и слава воину! 🏆\n", p1.Name)
	} else {
		fmt.Printf("🏆 ПОБЕДИТЕЛЬ: %s! Честь и слава воину! 🏆\n", p2.Name)
	}
	fmt.Println("══════════════════════════════════════════════════════════")
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

// ==================== СЕРВЕР ====================

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

// ==================== КЛИЕНТ ====================

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

// ==================== ОБЩАЯ ЛОГИКА БОЯ ====================

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
			// Делаем выбор
			fmt.Println("\n  ⚔  Выберите часть тела для АТАКИ:")
			fmt.Println("     1. Голова")
			fmt.Println("     2. Торс")
			fmt.Println("     3. Ноги")
			fmt.Print("  ➤ ")
			var attack int
			fmt.Scan(&attack)

			fmt.Println("\n  🛡  Выберите часть тела для ЗАЩИТЫ:")
			fmt.Println("     1. Голова")
			fmt.Println("     2. Торс")
			fmt.Println("     3. Ноги")
			fmt.Print("  ➤ ")
			var block int
			fmt.Scan(&block)

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

			var otherAction PlayerAction
			decoder.Decode(&otherAction)
			otherPlayer.SetAttack(getBodyPart(otherAction.Attack))
			otherPlayer.SetBlock(getBodyPart(otherAction.Block))

			// Делаем свой ход
			fmt.Println("\n  ⚔  Выберите часть тела для АТАКИ:")
			fmt.Println("     1. Голова")
			fmt.Println("     2. Торс")
			fmt.Println("     3. Ноги")
			fmt.Print("  ➤ ")
			var attack int
			fmt.Scan(&attack)

			fmt.Println("\n  🛡  Выберите часть тела для ЗАЩИТЫ:")
			fmt.Println("     1. Голова")
			fmt.Println("     2. Торс")
			fmt.Println("     3. Ноги")
			fmt.Print("  ➤ ")
			var block int
			fmt.Scan(&block)

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
	fmt.Println("║      ДОБРО ПОЖАЛОВАТЬ В TAD ADVENTURE           ║")
	fmt.Println("╠══════════════════════════════════════════════════╣")
	fmt.Println("║         Игра с фоновым чатом!                   ║")
	fmt.Println("║     💬 Общайтесь с другими игроками             ║")
	fmt.Println("║     🌐 Сетевой PvP режим                         ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()
}

func createPlayer() *Player {
	var playerName string
	fmt.Print("  Введите имя вашего персонажа: ")
	fmt.Scan(&playerName)

	// Очищаем буфер ввода после Scan
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	fmt.Println("\n╔══════════════════════════════════════════════════╗")
	fmt.Println("║              СОЗДАНИЕ ПЕРСОНАЖА                 ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Printf("\n  Добро пожаловать, %s!\n", playerName)
	fmt.Println("\n  Стартовые предметы:")
	fmt.Println("    ⚔ Деревянный меч         [АТК +3]")
	fmt.Println("    🛡 Кожаный жилет          [ЗАЩ +2]")
	fmt.Println("    💊 Простое зелье здоровья [ВОССТ +25 HP]")
	fmt.Println()

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

func createPlayerForPvP(playerNumber int) *Player {
	var playerName string
	fmt.Printf("  Введите имя игрока %d: ", playerNumber)
	fmt.Scan(&playerName)

	// Очищаем буфер ввода после Scan
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

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

func displayPrologue() {
	fmt.Println("\n╔══════════════════════════════════════════════════╗")
	fmt.Println("║                    ПРОЛОГ                        ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println("\n  В мире под названием ДэйВилл жил")
	fmt.Println("  маленький поросёнок по имени Тэд.")
	fmt.Println("  Он был не как все - его манила жажда")
	fmt.Println("  приключений и мечта о легендарных")
	fmt.Println("  сокровищах мира.")
	fmt.Println()
	fmt.Println("  Однажды, найдя старую карту в лесу,")
	fmt.Println("  Тэд решил: пора отправляться в путь!")
	fmt.Println("  Но дорога к сокровищам полна опасностей")
	fmt.Println("  и коварных врагов...")
	fmt.Println()
}

func displayEpilogue() {
	fmt.Println("\n╔══════════════════════════════════════════════════╗")
	fmt.Println("║                    ЭПИЛОГ                        ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println("\n  Пройдя через все испытания, Тэд наконец")
	fmt.Println("  достиг Запретного Храма.")
	fmt.Println("  Внутри его ждал не просто сундук с золотом,")
	fmt.Println("  а нечто большее - мудрость предков.")
	fmt.Println()
	fmt.Println("  Оказалось, что настоящее сокровище - это")
	fmt.Println("  путешествие, друзья, которых он приобрёл,")
	fmt.Println("  и уроки, которые он усвоил.")
	fmt.Println()
	fmt.Println("  Тэд вернулся в ДэйВилл героем, и его история")
	fmt.Println("  вдохновила многих молодых поросят на")
	fmt.Println("  собственные приключения!")
	fmt.Println()
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
		fmt.Println("  5. Выйти в главное меню")
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
	fmt.Println("  с переходом хода от одного игрока к другому.\n")

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

func startStoryMode() {
	displayPrologue()

	player := createPlayer()

	// Определяем врагов для каждого этапа путешествия с трофеями
	enemies := []struct {
		name     string
		hp       int
		strength int
		story    string
		loot     Item
	}{
		{
			name:     "Атомный-Кролик",
			hp:       60,
			strength: 10,
			story:    "В лесу Тэда внезапно остановил Атомный-Кролик! 'Живым ты отсюда не уйдешь!' - проскрипел он.",
			loot:     generateRandomItem(),
		},
		{
			name:     "Качок-Ворон",
			hp:       80,
			strength: 15,
			story:    "Переходя через Туманые Горы, Тэд столкнулся с Вороном-качком, который хотел задушить поросенка своими бицепсами!",
			loot:     generateRandomItem(),
		},
		{
			name:     "Медведь по кличке Самосвал",
			hp:       120,
			strength: 20,
			story:    "У входа в Запретный Храм Древних Свиней дорогу Тэду преградил грозный Медведь Самосвал - последний страж сокровищ!",
			loot:     generateRandomItem(),
		},
	}

	// Основной игровой цикл с сюжетными вставками
	for i, enemyInfo := range enemies {
		// Даем игроку возможность управлять инвентарем перед боем
		if i > 0 {
			fmt.Println("\n  Перед следующим боем вы можете проверить инвентарь")
			fmt.Print("  Хотите зайти в меню инвентаря? (1-да, 2-нет): ")
			var inventoryChoice int
			fmt.Scan(&inventoryChoice)
			if inventoryChoice == 1 {
				showInventoryMenu(player)
			}
		}

		fmt.Printf("\n╔══════════════════════════════════════════════════╗")
		fmt.Printf("\n║            ГЛАВА %d: %-20s      ║\n", i+1, enemyInfo.name)
		fmt.Printf("╚══════════════════════════════════════════════════╝\n")
		fmt.Printf("\n  %s\n", enemyInfo.story)
		fmt.Println()

		enemy := &Enemy{
			BaseCharacter: BaseCharacter{
				Name:     enemyInfo.name,
				HP:       enemyInfo.hp,
				MaxHP:    enemyInfo.hp,
				Strength: enemyInfo.strength,
			},
			Loot: enemyInfo.loot,
		}

		battle := NewBattle(player, enemy)
		playerWon := battle.Start()

		if !playerWon {
			fmt.Println("\n╔══════════════════════════════════════════════════╗")
			fmt.Println("║                  GAME OVER                       ║")
			fmt.Println("╚══════════════════════════════════════════════════╝")
			fmt.Println("\n  К сожалению, ваше приключение завершилось здесь...")
			fmt.Println("  Попробуйте снова!")
			return
		}

		// Восстанавливаем немного здоровья после боя (кроме последнего)
		if i < len(enemies)-1 {
			healAmount := 30
			player.HP += healAmount
			if player.HP > player.MaxHP {
				player.HP = player.MaxHP
			}
			fmt.Println("\n················································")
			fmt.Println("                    ОТДЫХ")
			fmt.Println("················································")
			fmt.Printf("\n  После боя вы нашли целебные травы\n")
			fmt.Printf("  и восстановили %d HP.\n", healAmount)
			showHealthBar(player.HP, player.MaxHP, "  Теперь у вас")
			fmt.Println()
		}

		// Пауза между боями
		if i < len(enemies)-1 {
			fmt.Print("  Нажмите Enter, чтобы продолжить...")
			fmt.Scanln()
		}
	}

	// Если игрок прошёл всех врагов - показываем эпилог
	displayEpilogue()

	// Показываем финальную статистику
	fmt.Println("\n╔══════════════════════════════════════════════════╗")
	fmt.Println("║                 ВАШИ ТРОФЕИ                      ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	player.ShowInventory()
	player.ShowEquipment()
}

func showMainMenu() {
	fmt.Println("\n╔══════════════════════════════════════════════════╗")
	fmt.Println("║                  ГЛАВНОЕ МЕНЮ                    ║")
	fmt.Println("╠══════════════════════════════════════════════════╣")
	fmt.Println("║  1. Сюжетная игра (Тэд поросёнок)               ║")
	fmt.Println("║  2. Сюжетная игра (Путь самурая)                ║")
	fmt.Println("║  3. PvP (Горячий стул) с чатом                   ║")
	fmt.Println("║  4. PvP (Дуэль самураев - упрощенный)           ║")
	fmt.Println("║  5. СЕТЕВОЙ PvP - создать сервер                 ║")
	fmt.Println("║  6. СЕТЕВОЙ PvP - подключиться к серверу         ║")
	fmt.Println("║  7. Выйти из игры                                ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Print("  ➤ ")
}

// ==================== ОСНОВНАЯ ФУНКЦИЯ ====================

func main() {
	rand.Seed(time.Now().UnixNano())

	// Запрашиваем имя для чата
	fmt.Print("\n  Введите ваше имя для чата: ")
	fmt.Scan(&userName)

	// Очищаем буфер ввода после Scan
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

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
			// Сюжетная игра (Тэд поросёнок)
			startStoryMode()
		case 2:
			// Сюжетная игра (Путь самурая)
			storyMode()
		case 3:
			// PvP (Горячий стул) с чатом
			startPvPMode()
		case 4:
			// PvP (Дуэль самураев - упрощенный)
			simplePvPMode()
		case 5:
			// Сетевой PvP - сервер
			startNetworkServer()
		case 6:
			// Сетевой PvP - клиент
			startNetworkClient()
		case 7:
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