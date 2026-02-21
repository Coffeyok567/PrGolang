package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	displayChan = make(chan string, 10)
	serverAddr  string
	userName    string
)

func main() {
	// Запрашиваем адрес сервера      https://ubiquitous-dollop-4jw7r4795qj53jjgp-8080.app.github.dev/
	fmt.Print("Введите адрес сервера (например, http://localhost:8080): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	serverAddr = scanner.Text()

	// Запрашиваем имя пользователя
	fmt.Print("Введите ваше имя: ")
	scanner.Scan()
	userName = scanner.Text()

	// Горутина для вывода сообщений
	go func() {
		for msg := range displayChan {
			fmt.Println(msg)
		}
	}()

	// Горутина для получения обновлений с сервера
	go receiveUpdates()

	// Главное меню
	for {
		fmt.Println("\n=== ГЛАВНОЕ МЕНЮ ===")
		fmt.Println("1. Написать сообщение в чат")
		fmt.Println("2. Играть в PvP режим")
		fmt.Println("3. Выход")
		fmt.Print("Выберите действие (1-3): ")

		scanner.Scan()
		choice := scanner.Text()

		switch choice {
		case "1":
			chatMode(scanner)
		case "2":
			pvpMode(scanner)
		case "3":
			fmt.Println("До свидания!")
			return
		default:
			fmt.Println("Неверный выбор, попробуйте снова")
		}
	}
}

func receiveUpdates() {
	lastCount := 0
	
	for {
		resp, err := http.Get(serverAddr)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		content := string(body)
		resp.Body.Close()

		// Проверяем, является ли ответ PvP командой
		if content == "ATTACK" || content == "DEFENSE" || content == "WAIT" || 
		   strings.Contains(content, "РЕЗУЛЬТАТ РАУНДА") {
			// Это PvP сообщение, пропускаем
			time.Sleep(1 * time.Second)
			continue
		}

		// Обрабатываем как чат сообщения
		lines := strings.Split(strings.TrimSpace(content), "\n")
		
		if len(lines) > lastCount && lines[0] != "" {
			for i := lastCount; i < len(lines); i++ {
				if lines[i] != "" {
					displayChan <- lines[i]
				}
			}
			lastCount = len(lines)
		}

		time.Sleep(2 * time.Second)
	}
}

func chatMode(scanner *bufio.Scanner) {
	fmt.Println("\n=== РЕЖИМ ЧАТА ===")
	fmt.Println("Введите сообщение (или 'menu' для возврата в меню):")
	
	for scanner.Scan() {
		message := scanner.Text()
		
		if message == "menu" {
			return
		}
		
		fullMessage := "[" + userName + "]: " + message
		http.Post(serverAddr, "text/plain", strings.NewReader(fullMessage))
		fmt.Println("Вы: " + message)
	}
}

func pvpMode(scanner *bufio.Scanner) {
	fmt.Println("\n=== PVP РЕЖИМ ===")
	
	// Регистрация в PvP режиме
	resp, err := http.Post(serverAddr, "text/plain", 
		strings.NewReader("register="+userName))
	if err != nil {
		fmt.Println("Ошибка подключения к серверу")
		return
	}
	
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	
	if string(body) == "SERVER_FULL" {
		fmt.Println("Сервер PvP режима заполнен (максимум 2 игрока)")
		return
	}
	
	fmt.Println("Регистрация успешна. Ожидание второго игрока...")
	
	// Основной цикл PvP
	for {
		resp, err := http.Get(serverAddr)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		state := string(body)
		resp.Body.Close()
		
		switch state {
		case "WAIT":
			fmt.Println("Ожидание второго игрока или его хода...")
			time.Sleep(2 * time.Second)
			
		case "ATTACK":
			fmt.Print("Выберите атаку (head/body/legs): ")
			scanner.Scan()
			attack := scanner.Text()
			http.Post(serverAddr, "text/plain",
				strings.NewReader("attack="+userName+":"+attack))
			fmt.Println("Атака отправлена!")
			
		case "DEFENSE":
			fmt.Print("Выберите защиту (head/body/legs): ")
			scanner.Scan()
			defense := scanner.Text()
			http.Post(serverAddr, "text/plain",
				strings.NewReader("defense="+userName+":"+defense))
			fmt.Println("Защита выбрана!")
			
		default:
			if strings.Contains(state, "РЕЗУЛЬТАТ РАУНДА") {
				fmt.Println("\n" + state)
				fmt.Println("\nНажмите Enter для продолжения...")
				scanner.Scan()
				
				// Проверяем, жив ли игрок
				if strings.Contains(state, userName+" = 0") {
					fmt.Println("Вы погибли! Возврат в меню...")
					return
				}
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}
}

