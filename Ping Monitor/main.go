package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// PingResult структура для результатов проверки
type PingResult struct {
	Host    string
	Success bool
	Latency time.Duration
	Attempt int
	Error   error
	Method  string // "TCP" или "ICMP"
}

func main() {
	// 1. Настройка флагов
	count := flag.Int("c", 1, "Количество повторов")
	isMonitor := flag.Bool("monitor", false, "Режим мониторинга")
	useICMP := flag.Bool("icmp", false, "Использовать ICMP ping (требуются права администратора)")
	fileFlag := flag.String("f", "", "Файл со списком хостов")
	flag.Parse()

	// Определяем имя файла (из флага -f или из первого аргумента)
	filename := *fileFlag
	if filename == "" {
		// Если флаг -f не использован, берем первый аргумент
		if flag.NArg() > 0 {
			filename = flag.Arg(0)
		}
	}

	if filename == "" {
		fmt.Println("Использование: go run main.go [-f файл] <файл> [-c N] [-monitor] [-icmp]")
		fmt.Println("  -f файл        файл со списком хостов")
		fmt.Println("  -c N           количество повторов (по умолчанию 1)")
		fmt.Println("  -monitor       режим мониторинга")
		fmt.Println("  -icmp          использовать ICMP ping (требуются права администратора)")
		fmt.Println("\nПримеры:")
		fmt.Println("  go run main.go hosts.txt -c 3")
		fmt.Println("  go run main.go -f hosts.txt -c 3 -monitor")
		return
	}

	fmt.Printf("Используется файл: %s\n", filename)
	fmt.Printf("Режим ICMP: %v\n", *useICMP)

	// 2. Чтение файла
	hosts, err := readLines(filename)
	if err != nil {
		fmt.Printf("Ошибка чтения файла: %v\n", err)
		return
	}

	if len(hosts) == 0 {
		fmt.Println("Файл не содержит хостов")
		return
	}

	fmt.Printf("Загружено хостов: %d\n", len(hosts))

	// Канал для сбора результатов
	results := make(chan PingResult)

	// Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Канал для сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// 3. Запуск воркеров
	for _, host := range hosts {
		wg.Add(1)
		go worker(ctx, &wg, host, *count, *isMonitor, *useICMP, results)
	}

	// 4. Слушатель результатов
	go func() {
		f, err := os.OpenFile("monitor.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Ошибка создания лог-файла: %v\n", err)
			return
		}
		defer f.Close()

		for result := range results {
			// Формируем строку вывода
			status := "OK"
			latency := result.Latency
			if !result.Success {
				status = "Timeout"
				latency = 0
			}

			line := fmt.Sprintf("%s | %-15s | %-7s | %v | %s | Попытка: %d",
				time.Now().Format("15:04:05"),
				result.Host,
				status,
				latency.Round(time.Millisecond),
				result.Method,
				result.Attempt)

			fmt.Println(line)
			f.WriteString(line + "\n")
		}
	}()

	// 5. Ожидание сигнала или завершения всех воркеров
	go func() {
		wg.Wait()
		close(results)
	}()

	<-sigChan
	fmt.Println("\nПолучен сигнал завершения...")
	cancel()

	// Даем время на завершение
	time.Sleep(time.Second)
	fmt.Println("Программа завершена")
}

// worker выполняет проверку хоста
func worker(ctx context.Context, wg *sync.WaitGroup, host string, count int, isMonitor bool, useICMP bool, results chan<- PingResult) {
	defer wg.Done()

	attempt := 1
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var result PingResult
			result.Host = host
			result.Attempt = attempt

			// Проверяем какой метод использовать
			if useICMP {
				result.Method = "ICMP"
				success, latency, err := pingICMP(host)
				result.Success = success
				result.Latency = latency
				result.Error = err
			} else {
				result.Method = "TCP"
				success, latency, err := pingTCP(host)
				result.Success = success
				result.Latency = latency
				result.Error = err
			}

			// Отправляем результат
			select {
			case results <- result:
			case <-ctx.Done():
				return
			}

			// Проверяем условия выхода
			if !isMonitor && attempt >= count {
				return
			}

			attempt++

			// Пауза между проверками
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return
			}
		}
	}
}

// pingTCP проверяет доступность хоста через TCP
func pingTCP(host string) (bool, time.Duration, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", host+":80", 2*time.Second)
	duration := time.Since(start)

	if err != nil {
		return false, 0, err
	}
	defer conn.Close()
	return true, duration, nil
}

// pingICMP проверяет доступность хоста через ICMP
func pingICMP(host string) (bool, time.Duration, error) {
	// Разрешаем IP адрес хоста
	ips, err := net.LookupIP(host)
	if err != nil {
		return false, 0, fmt.Errorf("DNS lookup failed: %v", err)
	}

	if len(ips) == 0 {
		return false, 0, fmt.Errorf("no IP addresses found for host")
	}

	// Берем первый IPv4 адрес
	var targetIP net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			targetIP = ip
			break
		}
	}

	if targetIP == nil {
		return false, 0, fmt.Errorf("no IPv4 address found for host")
	}

	// Создаем ICMP соединение
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false, 0, fmt.Errorf("failed to create ICMP connection (нужны права администратора): %v", err)
	}
	defer conn.Close()

	// Создаем ICMP echo запрос
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("ping"),
		},
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return false, 0, err
	}

	// Отправляем запрос и замеряем время
	start := time.Now()

	_, err = conn.WriteTo(msgBytes, &net.IPAddr{IP: targetIP})
	if err != nil {
		return false, 0, err
	}

	// Читаем ответ с таймаутом
	reply := make([]byte, 1500)
	err = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err != nil {
		return false, 0, err
	}

	n, peer, err := conn.ReadFrom(reply)
	if err != nil {
		return false, 0, err
	}

	duration := time.Since(start)

	// Парсим ответ
	replyMsg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply[:n])
	if err != nil {
		return false, 0, err
	}

	// Проверяем тип ответа
	switch replyMsg.Type {
	case ipv4.ICMPTypeEchoReply:
		// Проверяем, что ответ от ожидаемого хоста
		if peer.String() == targetIP.String() {
			return true, duration, nil
		}
		return false, 0, fmt.Errorf("received reply from unexpected host: %s", peer)
	default:
		return false, 0, fmt.Errorf("received unexpected ICMP type: %v", replyMsg.Type)
	}
}

// readLines читает файл и возвращает список строк
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" { // Пропускаем пустые строки
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}
