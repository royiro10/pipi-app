package main

import (
	"context"
	"log"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type PipiMessageHandler struct {
	pipiSessions   map[string]*Pipi
	whatsappClient *WhatsappClient
}

func NewMessageHandler(whatsappClient *WhatsappClient) *PipiMessageHandler {
	messageHandler := &PipiMessageHandler{
		pipiSessions:   make(map[string]*Pipi),
		whatsappClient: whatsappClient,
	}

	return messageHandler
}

func (messageHandler *PipiMessageHandler) Handler() whatsmeow.EventHandler {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			log.Default().Println("Recived message!")
			messageHandler.handleMessage(v)
		}
	}
}

func (messageHandler *PipiMessageHandler) handleMessage(msg *events.Message) {
	if chatJid := msg.Info.Chat; chatJid.String() != "" {
		if chatJid.String() != "972523236663@s.whatsapp.net" {
			log.Default().Println("ignore message from: ", chatJid.String())
			return
		}

		log.Default().Println("Recive message message from: ", chatJid.String())

		ctx := context.TODO()

		session := messageHandler.pipiSessions[chatJid.String()]
		if session == nil {
			session = NewPipi()
			messageHandler.pipiSessions[chatJid.String()] = session
		}

		log.Default().Printf("Received a message: %s\n", msg.Message.GetConversation())

		msgContent := msg.Message.GetConversation()
		pipiResponse, err := session.SendMessage(ctx, msgContent)
		if err != nil {
			log.Default().Panicf("could not create response from pipi", err)
		}

		err = messageHandler.whatsappClient.SendText(chatJid, pipiResponse)
		if err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}
	}
}
