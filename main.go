package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
)

var SystemPromptPath string = "./secrets/system_prompt.txt"

func JoinWithBaseDir(paths ...string) string {
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exeDir := filepath.Dir(exe)
	paths = append([]string{exeDir}, paths...)
	jointPath := filepath.Join(paths...)
	return jointPath
}

func GetSystemPrompt() string {
	data, err := os.ReadFile(JoinWithBaseDir(SystemPromptPath))
	if err != nil {
		fmt.Println("Error reading file:", err)
		return ""
	}

	return string(data)
}

func main() {
	fmt.Println("Hello world")

	whatsappClient := NewWhatsappClient()
	if err := whatsappClient.Init(); err != nil {
		panic(err)
	}
	defer whatsappClient.Destory()

	log.Default().Println("whatsapp client is up")

	pipiMessageHandler := NewMessageHandler(whatsappClient)
	whatsappClient.AddEventHandler(pipiMessageHandler.Handler())

	log.Default().Println("up and running")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Default().Println("shut down")
	if GlobalGenaiClient != nil {
		GlobalGenaiClient.Close()
	}
}
