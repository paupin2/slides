package main

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Screen interface {
	OnShutdown(func())
	Show(string)
}

type tScreen struct {
	deck     Deck
	conn     *websocket.Conn
	shutdown func()
	updates  chan string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 60 * time.Second
)

func (c *tScreen) OnShutdown(fn func()) {
	c.shutdown = fn
}

func (c *tScreen) Show(slide string) {
	c.updates <- slide
}

func (scr *tScreen) writer() {
	scrlog := log.With().
		Str("deck", scr.deck.Title).
		Str("client", scr.conn.RemoteAddr().String()).
		Logger()

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		scr.conn.Close()
		if scr.shutdown != nil {
			scr.shutdown()
		}
		scrlog.Debug().Msg("disconnected")
	}()

	type tMessage struct {
		Show string `json:"show"`
	}

	scrlog.Debug().Msg("connected")
	for {
		select {
		case slide, ok := <-scr.updates:
			scr.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				scr.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := scr.conn.WriteJSON(tMessage{slide}); err != nil {
				return
			}

		case <-ticker.C:
			scr.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := scr.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func handleSocketConnection(s *Server, d Deck, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("upgrading socket")
		return
	}
	screen := &tScreen{
		deck:    d,
		conn:    conn,
		updates: make(chan string, 10),
	}
	s.AddClient(d, screen)
	go screen.writer()
	screen.updates <- d.Slide
}
