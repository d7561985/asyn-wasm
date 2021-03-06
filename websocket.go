// +build js,wasm

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

type MsgCallBack func(string) (interface{}, error)

// WebSocket represent ws JavaScript adapter
// https://developer.mozilla.org/en-US/docs/Web/API/WebSocket
type WebSocket struct {
	// ws:// or wss://
	host    string
	promise *Promise
	ws      *js.Value

	// reconnect allow reconnect operation on close
	reconnect bool

	collBack MsgCallBack
}

// NewWebSocket create JS WebSocket adapter
// @host - URI with wss:// or ws:// format
// @reconnect - enable reconnection operation during close connection or during send attempt to closed connection
func NewWebSocket(host string, reconnect bool, cb MsgCallBack) *WebSocket {
	res := &WebSocket{host: host, reconnect: reconnect, collBack: defaultCallBack}

	if cb != nil {
		res.collBack = cb
	}

	return res
}

// Connect evaluate connection operation to provided host
// @return JS Promise where would be send connection result
func (w *WebSocket) Connect() js.Value {
	return w.open()
}

func (w *WebSocket) open() js.Value {
	if w.promise == nil {
		w.promise = NewPromise()
	}

	ws := js.Global().Get(jsref.JSGlobalClassWebSocket).New(w.host)

	ws.Call(jsref.AddEventListener, wsEventOpen, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.resolve(nil, true)
		return nil
	}))

	ws.Call(jsref.AddEventListener, wsEventMessage, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		jsBlob := args[0].Get("data")

		out, err := w.collBack(jsBlob.String())
		if err != nil {
			w.reject(err.Error(), true)
			return nil
		}

		w.resolve(out, true)
		return nil
	}))

	ws.Call(jsref.AddEventListener, wsEventClose, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if w.reconnect {
			w.open()

			return nil
		}

		w.reject("ws is closed", true)

		return nil
	}))

	ws.Call(jsref.AddEventListener, wsEventError, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if w.ReadyState() == WsOpen || w.reconnect == false {
			w.reject(fmt.Sprintf("ws error: [%v]", args[0].String()), true)
			return nil
		}

		println("WS error: ", args[0].String())

		return nil
	}))

	w.ws = &ws

	return w.promise.JSValue()
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

func defaultCallBack(in string) (interface{}, error) {
	var out interface{}
	if err := json.Unmarshal([]byte(in), &out); err != nil {
		return nil, err
	}

	return out, nil
}
