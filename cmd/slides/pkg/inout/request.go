package inout

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Ajax struct {
	Status int    `json:""`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Data   any    `json:"data,omitempty"`
}

func (ajax Ajax) Reply() *Reply {
	// encode as JSON
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(ajax); err != nil {
		log.Error().Err(err).Msg("encoding error")
		return Status(http.StatusInternalServerError)
	}

	reply := Reply{Status: http.StatusOK, Bytes: buf.Bytes()}
	if ajax.Status != 0 {
		reply.Status = ajax.Status
	}
	reply.Header("Content-Type", "application/json")
	return &reply
}

type Request struct {
	w         http.ResponseWriter
	r         *http.Request
	forceJSON bool
	status    int
	err       error
	vals      url.Values
}

func NewRequest(w http.ResponseWriter, r *http.Request) *Request {
	return &Request{
		w: w,
		r: r,
	}
}

func (req *Request) Path() string {
	return req.r.URL.Path
}

func (req *Request) Read(d any) error {
	return json.NewDecoder(req.r.Body).Decode(d)
}

func (req *Request) IsAjax() {
	req.forceJSON = true
}

func (req *Request) AcceptsGzip() bool {
	accept := req.r.Header.Get("Accept-Encoding")
	return strings.Contains(accept, "gzip")
}

func (req *Request) Method() string {
	return req.r.Method
}

func (req *Request) Failed() bool {
	return req.err != nil
}

func (req *Request) Error(format string, a ...any) {
	if !req.Failed() {
		req.err = fmt.Errorf(format, a...)
	}
}

func (req *Request) Send(reply *Reply) {
	if req.status == 0 {
		// default status to OK
		req.status = http.StatusOK
	}

	if reply == nil {
		reply = &Reply{Status: http.StatusOK}
	}

	if req.Failed() {
		// previous errors take precedence over reply
		if req.status >= 200 && req.status < 400 {
			// ensure we have a bad status
			req.status = http.StatusInternalServerError
		}

		if req.forceJSON {
			// force reply to be JSON
			reply = JSON(req.err)
		} else {
			// regular request, send error as a string
			reply = &Reply{
				Status: req.status,
				Bytes:  []byte(req.err.Error()),
			}
		}
	}

	// set headers
	if len(reply.Bytes) > 0 {
		reply.Header("Content-Length", "%d", len(reply.Bytes))
	}
	if len(reply.headers) > 0 {
		h := req.w.Header()
		for key, values := range reply.headers {
			h[key] = values
		}
	}
	req.w.WriteHeader(req.status)

	// write bytes
	if len(reply.Bytes) > 0 {
		_, _ = req.w.Write(reply.Bytes)
	}

	log.Info().
		Int("status", req.status).
		Str("method", req.r.Method).
		Str("path", req.r.URL.Path).
		Int("reply-length", len(reply.Bytes)).
		Msg("request")
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (req *Request) Upgrade() (*websocket.Conn, error) {
	return upgrader.Upgrade(req.w, req.r, nil)
}

type param struct {
	name  string
	req   *Request
	vals  []string
	err   error
	valid bool
}

func (p *param) error(format string, a ...any) {
	if p.req.err == nil {
		p.req.status = http.StatusBadRequest
		p.req.Error(format, a...)
	}
}

type StrParam struct {
	param
	value string
	def   *string
}

func (req *Request) param(name string) (p param) {
	if req.vals == nil {
		req.vals = req.r.URL.Query()
	}

	return param{name: name, req: req, vals: req.vals[name]}
}

func (req *Request) Str(name string) (p *StrParam) {
	return &StrParam{param: req.param(name)}
}

func (p *StrParam) Def(v string) *StrParam {
	p.def = &v
	return p
}

func (p *StrParam) Get() string {
	switch len(p.vals) {
	case 0:
		if p.def != nil {
			return *p.def
		}
		p.error(`missing param "%s"`, p.name)
	case 1:
		return p.vals[0]
	default:
		p.error(`multiple values for "%s"`, p.name)
	}
	return ""
}

type IntParam struct {
	param
	value int
	def   *int
}

func (req *Request) Int(name string) (p *IntParam) {
	return &IntParam{param: req.param(name)}
}

func (p *IntParam) Def(v int) *IntParam {
	p.def = &v
	return p
}

func (p *IntParam) Get() int {
	switch len(p.vals) {
	case 0:
		if p.def != nil {
			return *p.def
		}
		p.error(`missing param "%s"`, p.name)
	case 1:
		i, err := strconv.Atoi(p.vals[0])
		if err != nil {
			p.error(`bad value for "%s": "%s"`, p.name, p.vals[0])
		}
		return i
	default:
		p.error(`multiple values for "%s"`, p.name)
	}
	return 0
}
