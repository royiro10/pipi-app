package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
)

type ServiceStatusVal string

const (
	SERVICE_STATUS_UP     ServiceStatusVal = "ok"
	SERVICE_STATUS_DOWN   ServiceStatusVal = "down"
	SERVICE_STATUS_ERR    ServiceStatusVal = "error"
	SERVICE_STATUS_UNKOWN ServiceStatusVal = "unknown"
)

type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type LogMessage struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

var WhatsappClientServiceName = "Whatsapp Client"
var PipiServiceName = "Gemini Service"

type ServerDependencies struct {
	WaClient *WhatsappClient
}

type Server struct {
	ctx    context.Context
	cancel context.CancelFunc

	logChannel     chan *LogMessage
	logHistory     []*LogMessage
	logListeners   map[uint32]func(*LogMessage)
	logListenersMu sync.Mutex

	serviceStatus map[string]ServiceStatus

	httpServer *http.Server
	QrBase64   string
}

func NewServer() *Server {
	s := &Server{
		logChannel:   make(chan *LogMessage),
		logHistory:   make([]*LogMessage, 0),
		logListeners: make(map[uint32]func(*LogMessage), 0),

		serviceStatus: map[string]ServiceStatus{
			"Api": {Name: "Api", Status: string(SERVICE_STATUS_UP)},
		},
	}

	s.ctx, s.cancel = context.WithCancel(context.TODO())

	s.addLogListener(func(lm *LogMessage) {
		s.logHistory = append(s.logHistory, lm)
	})

	go s.listenToLogs()
	return s
}

func (s *Server) Listen(addr string) {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./static"))

	mux.HandleFunc("/api/qr", s.QrHandler)
	mux.HandleFunc("/api/stop", s.StopHandler)
	mux.HandleFunc("/api/logs", s.LogsHandler)
	mux.HandleFunc("/api/logs/history", s.LogsHistoryHandler)
	mux.HandleFunc("/api/services-status", s.ServicesStatusHandler)
	mux.Handle("/", fs)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Generate log messages in a goroutine
	// go func() {
	// 	select {
	// 	case <-s.ctx.Done():
	// 		return
	// 	default:
	// 		for i := 0; ; i++ {
	// 			logMessage := &LogMessage{
	// 				Timestamp: fmt.Sprintf("[%d]", i),
	// 				Message:   fmt.Sprintf("Sample log message number %d", i),
	// 			}
	// 			s.logChannel <- logMessage
	// 			time.Sleep(time.Second * 2)
	// 		}
	// 	}

	// }()

	log.Printf("Starting server on %s", addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", addr, err)
	}
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	s.cancel()
}

func (s *Server) LogsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher := w.(http.Flusher)
	flushLogMessage := func(lm *LogMessage) {
		data, err := json.Marshal(lm)
		if err != nil {
			log.Println("Error marshalling log message:", err)
			return
		}

		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush() // Flush data to the client
	}

	for _, lm := range s.logHistory {
		flushLogMessage(lm)
	}

	listenerId := s.addLogListener(flushLogMessage)
	defer s.removeLogListener(listenerId)

	<-r.Context().Done()
	log.Println("Client request closed")
}

func (s *Server) LogsHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.logHistory)
}

func (s *Server) ServicesStatusHandler(w http.ResponseWriter, r *http.Request) {
	v := make([]ServiceStatus, len(s.serviceStatus), len(s.serviceStatus))
	idx := 0
	for _, serviceStatus := range s.serviceStatus {
		v[idx] = serviceStatus
		idx++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) StopHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Server is shutting down..."))

	log.Default().Println("stopping...")
	// TODO: send stop to all services
}

func (s *Server) QrHandler(w http.ResponseWriter, r *http.Request) {
	if waClientStatus, ok := s.serviceStatus[WhatsappClientServiceName]; ok && waClientStatus.Status == string(SERVICE_STATUS_UP) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	qrCode, err := qrcode.Encode(s.QrBase64, qrcode.Medium, 256)
	if err != nil {
		log.Println("Error generating QR code:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(qrCode)
}

func (s *Server) MakeServiceNotifier(service string) func(status ServiceStatusVal) {
	var latestSatatus ServiceStatusVal = ""

	return func(status ServiceStatusVal) {
		if latestSatatus == status {
			return
		}

		log.Default().Printf("[%s] status -> %s", service, status)
		s.UpdateServiceStatus(service, status)

		latestSatatus = status
	}
}

func (s *Server) UpdateServiceStatus(service string, status ServiceStatusVal) {
	s.serviceStatus[service] = ServiceStatus{
		Name:   service,
		Status: string(status),
	}
}

func (s *Server) listenToLogs() {
	for {
		select {
		case message, ok := <-s.logChannel:
			if !ok {
				log.Println("Log channel closed")
				return
			}

			for _, listener := range s.logListeners {
				listener(message)
			}
		case <-s.ctx.Done():
			log.Println("Server ctx canceled")
			return
		}
	}
}

func (s *Server) addLogListener(listener func(*LogMessage)) uint32 {
	s.logListenersMu.Lock()
	defer s.logListenersMu.Unlock()

	listenerId := uint32(time.Now().Unix())
	s.logListeners[listenerId] = listener

	return listenerId
}

func (s *Server) removeLogListener(listenerId uint32) {
	s.logListenersMu.Lock()
	defer s.logListenersMu.Unlock()

	delete(s.logListeners, listenerId)
}
