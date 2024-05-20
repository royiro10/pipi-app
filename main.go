package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var SystemPromptPath string = "./secrets/_ai_system_prompt.txt"
var AllowedJidsPath string = "./secrets/allowed_jids.json"

func main() {
	log.Default().Println("starting pipi")

	bouncer := NewBouncer(AllowedJidsPath)

	server := NewServer()
	server.Bouncer = bouncer
	releaseStdoutLeach := listenToStdout(server.logChannel)
	defer releaseStdoutLeach()

	go server.Listen(":80")

	waClientStatusNotifier := server.MakeServiceNotifier(WhatsappClientServiceName)
	pipiStatusNotifier := server.MakeServiceNotifier(PipiServiceName)

	setWhatsappQr := func(qrBase64 string) {
		server.QrBase64 = qrBase64
	}

	whatsappClient := NewWhatsappClient(waClientStatusNotifier, setWhatsappQr)
	if err := whatsappClient.Init(); err != nil {
		panic(err)
	}
	defer whatsappClient.Destory()

	log.Default().Println("whatsapp client is up")

	pipiMessageHandler := NewMessageHandler(whatsappClient, bouncer, pipiStatusNotifier)
	whatsappClient.AddEventHandler(pipiMessageHandler.Handler())

	log.Default().Println("up and running")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Default().Println("shut down")

	server.Stop()
	if GlobalGenaiClient != nil {
		GlobalGenaiClient.Close()
	}
}

func listenToStdout(outputChan chan *LogMessage) func() {
	chanReader, chanWriter, _ := os.Pipe()

	// save existing stdout | MultiWriter writes to saved stdout and file
	out := os.Stdout
	mw := io.MultiWriter(out, chanWriter)

	// get pipe reader and writer | writes to pipe writer come out pipe reader
	r, w, _ := os.Pipe()

	// replace stdout,stderr with pipe writer | all writes to stdout, stderr will go through pipe instead (fmt.print, log)
	os.Stdout = w
	os.Stderr = w

	// writes with log.Print should also write to mw
	log.SetOutput(mw)

	//create channel to control exit | will block until all copies are finished
	exit := make(chan bool)

	go func() {

		buf := make([]byte, 1024)
		for {
			n, err := chanReader.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				panic(err)
			}
			outputChan <- &LogMessage{
				Timestamp: time.Now().Format("2006-01-02 15:04:05"),
				Message:   string(buf[:n]),
			}
		}
	}()

	go func() {
		// copy all reads from pipe to multiwriter, which writes to stdout and file
		_, _ = io.Copy(mw, r)
		// when r or w is closed copy will finish and true will be sent to channel
		exit <- true
	}()

	release := func() {
		// close writer then block on exit channel | this will let mw finish writing before the program exits
		_ = w.Close()
		<-exit
		// close file after all writes have finished
		_ = chanWriter.Close()
	}

	return release
}

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
