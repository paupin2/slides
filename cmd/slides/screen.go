package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/paupin2/slides/cmd/slides/pkg/inout"
	"github.com/paupin2/slides/pkg/data"
	"github.com/rs/zerolog/log"
)

type Content struct {
	Text string `json:"text"`
}

type Screen struct {
	title    string
	conn     *websocket.Conn
	shutdown func()
	updates  chan []byte
}

func (srv *Server) HandleScreen(req *inout.Request) *inout.Reply {
	title := req.Str("title").Get()
	if req.Failed() {
		return inout.Error(http.StatusBadRequest, "bad title")
	}
	if err := data.CheckTitle(title); err != nil {
		return inout.Error(http.StatusBadRequest, "bad title")
	}

	conn, err := req.Upgrade()
	if err != nil {
		log.Error().Err(err).Msg("upgrading socket")
		return inout.Error(http.StatusInternalServerError, "error connecting")
	}

	screen := &Screen{
		title:   title,
		conn:    conn,
		updates: make(chan []byte, 10),
		shutdown: func() {
			// remove screen when the connection shuts down
			srv.lock.Lock()
			defer srv.lock.Unlock()
			scrs := srv.screens[title]
			for i, cl := range scrs {
				if cl.conn == conn {
					// remove screen
					srv.screens[title] = append(scrs[:i], scrs[i+1:]...)
					return
				}
			}
			return
		},
	}

	// add the screen
	srv.lock.Lock()
	srv.screens[title] = append(srv.screens[title], screen)
	srv.lock.Unlock()

	// start writer, send current slide (if any)
	go screen.writer()
	_ = screen.SendJSON(Content{srv.get(title)})
	return inout.Status(http.StatusOK)
}

// Broadcast the data to all of this deck's screeens
func (s *Server) Broadcast(title string, data any) {
	// encode data if necessary
	var msg []byte
	if b, isBytes := data.([]byte); isBytes {
		msg = b
	} else {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(data); err != nil {
			// don't send
			return
		}
		msg = buf.Bytes()
	}

	if len(msg) == 0 {
		// don't send empty data
		return
	}

	// get list of screens
	s.lock.Lock()
	screens := s.screens[title]
	ls := make([]*Screen, len(screens))
	for i, s := range screens {
		ls[i] = s
	}
	s.lock.Unlock()

	// broadcast to all screens
	for _, s := range ls {
		s.Send(msg)
	}
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 60 * time.Second
)

func (c *Screen) OnShutdown(fn func()) {
	c.shutdown = fn
}

func (c *Screen) Send(msg []byte) {
	c.updates <- msg
}

func (c *Screen) SendJSON(data any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return err
	}
	c.updates <- buf.Bytes()
	return nil
}

func (scr *Screen) writer() {
	scrlog := log.With().
		Str("deck", scr.title).
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

	scrlog.Debug().Msg("connected")
	for {
		select {
		case update, ok := <-scr.updates:
			_ = scr.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = scr.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := scr.conn.WriteMessage(websocket.TextMessage, update); err != nil {
				return
			}

		case <-ticker.C:
			_ = scr.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := scr.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
