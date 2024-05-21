package main

import (
	"context"
	"log"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type PipiMessageHandler struct {
	pipiSessions          map[string]*Pipi
	whatsappClient        *WhatsappClient
	serviceStatusNotifier func(status ServiceStatusVal, reason string)
	bouncer               *Bouncer
}

func NewMessageHandler(whatsappClient *WhatsappClient, bouncer *Bouncer, serviceStatusNotifier func(status ServiceStatusVal, reason string)) *PipiMessageHandler {
	messageHandler := &PipiMessageHandler{
		pipiSessions:          make(map[string]*Pipi),
		whatsappClient:        whatsappClient,
		serviceStatusNotifier: serviceStatusNotifier,
		bouncer:               bouncer,
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
		if !messageHandler.bouncer.isAllowd(chatJid.String()) {
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

		msgContent := getMessageContent(msg)
		if msgContent == "" {
			return
		}

		log.Default().Printf("Received a message: %s\n", msgContent)

		if ResetCode == msgContent {
			log.Default().Println("reset request at:", chatJid.String())
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

func getMessageContent(msg *events.Message) string {
	switch {
	case msg.Message.Conversation != nil:
		return msg.Message.GetConversation()

	case msg.Message.ExtendedTextMessage != nil:
		return msg.Message.GetExtendedTextMessage().GetText()

	default:
		log.Default().Println("ERROR: could not extract conent", msg)
		return ""
	}
}
