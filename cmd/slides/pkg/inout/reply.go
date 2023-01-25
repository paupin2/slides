package inout

import (
	"fmt"
	"net/http"
)

type Reply struct {
	Status  int
	Bytes   []byte
	headers http.Header
}

func (r *Reply) Header(key, value string, args ...any) {
	if r.headers == nil {
		r.headers = http.Header{}
	}
	if len(args) > 0 {
		value = fmt.Sprintf(value, args...)
	}
	r.headers.Set(key, value)
}

func Status(code int) *Reply {
	return &Reply{
		Status: code,
		Bytes:  []byte(http.StatusText(code)),
	}
}

func Error(code int, msg string, a ...any) *Reply {
	if len(a) > 0 {
		msg = fmt.Sprintf(msg, a...)
	}
	ajax := &Ajax{Status: code, OK: false, Error: msg}
	if msg == "" {
		ajax.Error = http.StatusText(code)
	}
	return ajax.Reply()
}

func Static(ctype string, content []byte) *Reply {
	r := Reply{
		Status: http.StatusOK,
		Bytes:  content,
	}
	r.Header("Content-Type", ctype)
	return &r
}

func JSON(data any) *Reply {
	var ajax Ajax
	if err, iserr := data.(error); iserr {
		ajax.Error = err.Error()
	} else {
		ajax.OK = true
		ajax.Data = data
	}
	return ajax.Reply()
}

func OK() *Reply {
	return Ajax{OK: true}.Reply()
}
