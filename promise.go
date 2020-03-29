// +build js,wasm

package asyn_wasm

import (
	"syscall/js"

	"github.com/d7561985/asyn-wasm/jsref"
)

// Promise implement js.Promise
type Promise struct {
	p       js.Value
	resolve js.Value
	reject  js.Value
}

func NewPromise() *Promise {
	out := &Promise{}

	out.p = js.Global().Get(jsref.JSGlobalClassPromise).New(js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		out.resolve, out.reject = args[0], args[1]
		return nil
	}))

	return out
}

// resolve promise
func (p *Promise) Resolve(res interface{}) {
	p.resolve.Invoke(res)
}

// reject promise
func (p *Promise) Reject(res interface{}) {
	p.reject.Invoke(res)
}

// JSValue is actual js Promise object which should be returned by
// js.Func into real js runtime
func (p *Promise) JSValue() js.Value {
	return p.p
}
