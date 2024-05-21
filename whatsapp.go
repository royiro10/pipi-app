package main

import (
	"context"
	"fmt"
	"os"
	"time"

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
	waClient              *whatsmeow.Client
	serviceStatusNotifier func(status ServiceStatusVal, reason string)
	notifyQr              func(string)
}

func NewWhatsappClient(serviceStatusNotifier func(status ServiceStatusVal, reason string), notifyQr func(string)) *WhatsappClient {
	client := &WhatsappClient{
		serviceStatusNotifier: serviceStatusNotifier,
		notifyQr:              notifyQr,
	}

	serviceStatusNotifier(SERVICE_STATUS_UNKOWN, "service initiaited")
	deviceStore, err := makeStore()
	if err != nil {
		serviceStatusNotifier(SERVICE_STATUS_ERR, err.Error())
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client.waClient = whatsmeow.NewClient(deviceStore, clientLog)

	return client

}

func (client *WhatsappClient) Init() error {

	// polling for status
	go func() {
		var PollingIntervalSeconds int64 = 60 * 5
		var PollingIntervalWhenDoneSeconds int64 = 10
		lastStatus := SERVICE_STATUS_DOWN

		time.Sleep(time.Duration(PollingIntervalWhenDoneSeconds * int64(time.Second)))

		for {

			if client.waClient.IsConnected() && client.waClient.IsLoggedIn() {
				client.serviceStatusNotifier(SERVICE_STATUS_UP, "client is connected and logged in")
				lastStatus = SERVICE_STATUS_UP
			} else {
				client.serviceStatusNotifier(SERVICE_STATUS_DOWN, "client is not connected or logged in")
				lastStatus = SERVICE_STATUS_DOWN
			}

			if lastStatus == SERVICE_STATUS_DOWN {
				time.Sleep(time.Duration(PollingIntervalWhenDoneSeconds * int64(time.Second)))
			} else {
				time.Sleep(time.Duration(PollingIntervalSeconds * int64(time.Second)))
			}
		}
	}()

	if client.waClient.Store.ID != nil {
		return client.waClient.Connect()
	}

	return client.firstTimeConnect()
}

func (client *WhatsappClient) Destory() {
	client.waClient.Disconnect()
	client.serviceStatusNotifier(SERVICE_STATUS_DOWN, "whatsapp client has been destoryed")
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
			client.notifyQr(evt.Code)
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
