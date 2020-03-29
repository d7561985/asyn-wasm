package asyn_wasm

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/d7561985/asyn-wasm/jsref"
)

const (
	wsEventOpen    = "open"
	wsEventMessage = "message"
	wsEventClose   = "close"
	wsEventError   = "error"
)

const (
	wsFieldReadyState = "readyState"
)

const (
	wsFunctionSend = "send"
)

const (
	WsConnecting = 0
	WsOpen       = 1
	WsClosing    = 2
	WsClosed     = 3
)

// WebSocket represent ws JavaScript adapter
// https://developer.mozilla.org/en-US/docs/Web/API/WebSocket
type WebSocket struct {
	// ws:// or wss://
	host    string
	promise *Promise
	ws      *js.Value

	// reconnect allow reconnect operation on close
	reconnect bool
}

// NewWebSocket create JS WebSocket adapter
// @host - URI with wss:// or ws:// format
// @reconnect - enable reconnection operation during close connection or during send attempt to closed connection
func NewWebSocket(host string, reconnect bool) *WebSocket {
	return &WebSocket{host: host, reconnect: reconnect}
}

// Connect evaluate connection operation to provided host
// @return JS Promise where would be send connection result
func (w *WebSocket) Connect() js.Value {
	return w.open()
}

func (w *WebSocket) open() js.Value {
	conPromise := NewPromise()
	ws := js.Global().Get(jsref.JSGlobalClassWebSocket).New(w.host)

	ws.Call(jsref.AddEventListener, wsEventOpen, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		conPromise.Resolve(nil)
		conPromise = nil

		return nil
	}))

	ws.Call(jsref.AddEventListener, wsEventMessage, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		jsBlob := args[0].Get("data")

		var out interface{}
		if err := json.Unmarshal([]byte(jsBlob.String()), &out); err != nil {
			w.reject(err.Error(), true)
			return nil
		}

		w.resolve(out, true)

		return nil
	}))

	ws.Call(jsref.AddEventListener, wsEventClose, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.reject(fmt.Errorf("ws is closed"), true)

		if w.reconnect {
			w.open()
		}

		return nil
	}))

	ws.Call(jsref.AddEventListener, wsEventError, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.reject(fmt.Errorf("ws error: [%v]", args), true)

		if conPromise != nil {
			conPromise.Reject(fmt.Errorf("ws error: [%v]", args))
			conPromise = nil
		}

		return nil
	}))

	w.ws = &ws
	w.promise = nil

	return conPromise.JSValue()
}

func (w *WebSocket) ReadyState() int {
	if w.ws == nil {
		return -1
	}

	return w.ws.Get(wsFieldReadyState).Int()
}

// @return JS Promise where would be putted further response
func (w *WebSocket) Send(msg string) js.Value {
	status, promise := w.ReadyState(), NewPromise()

	switch {
	case w.promise != nil:
		promise.Reject("already exist request")
	case status == WsOpen:
		w.promise = promise
		w.ws.Call(wsFunctionSend, js.ValueOf(msg))
	case status == WsClosed || status == WsClosing:
		promise.Reject("ws closed")

		// ToDo: we can save our message and link to the promise of open connection
		// but here should add support async send message or something similar.
		if w.reconnect {
			_ = w.open()
		}
	case status == WsConnecting:
		promise.Reject("ws has status connecting...")
	}

	return promise.JSValue()
}

func (w *WebSocket) resolve(result interface{}, clean bool) {
	if w.promise == nil {
		return
	}

	w.promise.Resolve(result)

	if clean {
		w.promise = nil
	}
}

func (w *WebSocket) reject(reason interface{}, clean bool) {
	if w.promise == nil {
		return
	}

	w.promise.Reject(reason)

	if clean {
		w.promise = nil
	}
}
