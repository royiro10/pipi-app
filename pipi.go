package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var GEMINAI_API_KEY string
var GlobalGenaiClient *genai.Client = nil

func setupGlobalGenaiClient() {
	if GlobalGenaiClient != nil {
		log.Default().Println("GlobalGenaiClient is already setup")
	}

	client, err := genai.NewClient(context.Background(), option.WithAPIKey(os.Getenv("GEMINAI_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}

	GlobalGenaiClient = client
}

type Pipi struct {
	chatSession *genai.ChatSession
}

func NewPipi() *Pipi {
	pipi := &Pipi{}

	if GlobalGenaiClient == nil {
		setupGlobalGenaiClient()
	}

	model := GlobalGenaiClient.GenerativeModel("gemini-1.0-pro")
	pipi.chatSession = model.StartChat()
	pipi.chatSession.History = []*genai.Content{
		{
			Parts: []genai.Part{
				genai.Text(GetSystemPrompt()),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("understood"),
			},
			Role: "model",
		},
	}

	return pipi
}

var MaxSendRetries = 10

func (p *Pipi) SendMessage(ctx context.Context, msg string) (string, error) {
	var err error = nil
	var response string
	for i := 0; i < MaxSendRetries && (err != nil || i == 0); i++ {
		log.Default().Println("try get response", i)

		response, err = p.sendMesssage(ctx, msg)
		if err == nil {
			return response, nil
		}
	}

	return "", err
}

func (p *Pipi) sendMesssage(ctx context.Context, msg string) (string, error) {
	resp, err := p.chatSession.SendMessage(ctx, genai.Text(msg))
	if err != nil {
		log.Default().Println("gemini send message failed:", err)
		return "", err
	}

	response := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	return response, nil
}
