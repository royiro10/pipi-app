package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type WhatsappClient struct {
	waClient *whatsmeow.Client
}

func NewWhatsappClient() *WhatsappClient {
	client := &WhatsappClient{}

	deviceStore, err := makeStore()
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client.waClient = whatsmeow.NewClient(deviceStore, clientLog)

	return client

}

func (client *WhatsappClient) Init() error {
	if client.waClient.Store.ID != nil {
		return client.waClient.Connect()
	}

	return client.firstTimeConnect()
}

func (client *WhatsappClient) Destory() {
	client.waClient.Disconnect()
}

func (client *WhatsappClient) AddEventHandler(eventHandler whatsmeow.EventHandler) uint32 {
	return client.waClient.AddEventHandler(eventHandler)
}

func (client *WhatsappClient) SendText(to types.JID, msg string) error {
	_, err := client.waClient.SendMessage(context.Background(), to, &waProto.Message{
		Conversation: proto.String(msg),
	})

	return err
}

func (client *WhatsappClient) firstTimeConnect() error {
	qrChan, _ := client.waClient.GetQRChannel(context.Background())
	err := client.waClient.Connect()
	if err != nil {
		return err
	}

	for evt := range qrChan {
		if evt.Event == "code" {
			fmt.Println("QR code:", evt.Code)
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
		} else {
			fmt.Println("Login event:", evt.Event)
		}
	}

	return nil
}

func makeStore() (*store.Device, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	fileAddr := "file:examplestore.db?_foreign_keys=on"

	container, err := sqlstore.New("sqlite3", fileAddr, dbLog)
	if err != nil {
		return nil, err
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, err
	}

	return deviceStore, nil
}
