package router

import (
	"github.com/gascore/dom"
	"github.com/gascore/dom/js"
	"github.com/gascore/gas"
)

func (ctx *Ctx) CustomPush(path string, replace bool) {
	if ctx.Settings.GetUserConfirmation != nil && ctx.Settings.GetUserConfirmation() {
		return
	}

	ctx.ChangeRoute(path, replace)

	dom.GetWindow().DispatchEvent(js.New("Event", ChangeRouteEvent))
}

func (ctx *Ctx) CustomPushDynamic(name string, params, queries gas.Map, replace bool) {
	ctx.CustomPush(ctx.fillPath(name, params, queries), replace)
}

// Push push user to another page
func (ctx *Ctx) Push(path string) {
	ctx.CustomPush(path, false)
}

// Replace replace current page
func (ctx *Ctx) Replace(path string) {
	ctx.CustomPush(path, true)
}

// PushDynamic push user to another route with params and queries
func (ctx *Ctx) PushDynamic(name string, params, queries gas.Map) {
	ctx.CustomPushDynamic(name, params, queries, false)
}

// ReplaceDynamic replace current page with page generated from name, params and queries
func (ctx *Ctx) ReplaceDynamic(name string, params, queries gas.Map) {
	ctx.CustomPushDynamic(name, params, queries, true)
}

func (ctx Ctx) link(path string, push func(gas.Event), e gas.External) *gas.Element {
	var attrs gas.Map
	if e.Attrs == nil {
		attrs = make(gas.Map)
	} else {
		attrs = e.Attrs()
	}
	
	attrs["href"] = ctx.Settings.BaseName + path

	beforePush := func(event gas.Event) {
		push(event)
		event.Call("preventDefault")
	}

	return gas.NE(
		&gas.E{
			Tag: "a",
			Attrs: func() gas.Map {
				return attrs
			},
			Handlers: map[string]gas.Handler {
				"click":    beforePush,
				"keyup.13": beforePush,
				"keyup.32": beforePush,
			},
		},
		e.Body...)
}

// Link create link to route
func (ctx *Ctx) Link(to string, e gas.External) *gas.Element {
	return ctx.link(
		to,
		func(e gas.Event) {
			ctx.Push(to)
		},
		e)
}

//LinkWithParams create link to route with queries and params
func (ctx *Ctx) LinkWithParams(name string, params, queries gas.Map, e gas.External) *gas.Element {
	return ctx.link(
		ctx.fillPath(name, params, queries),
		func(e gas.Event) {
			ctx.PushDynamic(name, params, queries)
		},
		e)
}
