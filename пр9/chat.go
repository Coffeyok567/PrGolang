package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Введите имя: ")
	username, _ := reader.ReadString('\n')

	go receiveMessages()

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')

		appendMessage(username + ": " + text)
		gitPush()
	}
}

// дописываем сообщение в файл
func appendMessage(msg string) {
	file, _ := os.OpenFile("messages.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	file.WriteString(msg)
}

// отправка в GitHub
func gitPush() {
	exec.Command("git", "add", "messages.txt").Run()
	exec.Command("git", "commit", "-m", "new message").Run()
	exec.Command("git", "push").Run()
}

// получение сообщений
func receiveMessages() {
	for {
		exec.Command("git", "pull").Run()

		data, err := os.ReadFile("messages.txt")
		if err == nil {
			fmt.Println("\n--- ЧАТ ---")
			fmt.Print(string(data))
			fmt.Println("------------")
		}

		time.Sleep(5 * time.Second)
	}
}
