package main

import (
	"context"
	"log"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type PipiMessageHandler struct {
	pipiSessions          map[string]*Pipi
	whatsappClient        *WhatsappClient
	serviceStatusNotifier func(status ServiceStatusVal)
	allowedJids           []string
}

func NewMessageHandler(whatsappClient *WhatsappClient, allowedJids []string, serviceStatusNotifier func(status ServiceStatusVal)) *PipiMessageHandler {
	messageHandler := &PipiMessageHandler{
		pipiSessions:          make(map[string]*Pipi),
		whatsappClient:        whatsappClient,
		serviceStatusNotifier: serviceStatusNotifier,
		allowedJids:           allowedJids,
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

var ResetCode = "!"

func (messageHandler *PipiMessageHandler) handleMessage(msg *events.Message) {
	if chatJid := msg.Info.Chat; chatJid.String() != "" {
		if !messageHandler.IsJidAllowed(chatJid) {
			log.Default().Println("ignore message from: ", chatJid.String())
			return
		}

		log.Default().Println("Recive message message from: ", chatJid.String())

		ctx := context.TODO()

		session := messageHandler.pipiSessions[chatJid.String()]
		if session == nil {
			session = NewPipi(messageHandler.serviceStatusNotifier)
			messageHandler.pipiSessions[chatJid.String()] = session
		}

		log.Default().Printf("Received a message: %s\n", msg.Message.GetConversation())

		msgContent := msg.Message.GetConversation()
		if ResetCode == msgContent {
			session = NewPipi(messageHandler.serviceStatusNotifier)
			messageHandler.pipiSessions[chatJid.String()] = session
		}

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

func (messageHandler *PipiMessageHandler) IsJidAllowed(chatJid types.JID) bool {
	for _, jidStr := range messageHandler.allowedJids {
		if jidStr == chatJid.String() {
			return true
		}
	}

	return false
}
