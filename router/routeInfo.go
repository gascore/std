package router

import (
	"fmt"
	"strings"

	"github.com/gascore/dom"
	"github.com/gascore/dom/js"
	"github.com/gascore/gas"
)

// RouteInfo info about current route for router
type RouteInfo struct {
	Name string
	URL  string

	Params      map[string]string // /links/:foo => {"foo": "bar"}
	QueryParams map[string]string // /links?foo=bar => {"foo": "bar"}

	Route Route

	Ctx *Ctx
}

// Push push user to another route
func (i RouteInfo) Push(path string, replace bool) {
	i.Ctx.Push(path, replace)
}

// Push push user to another page
func (ctx *Ctx) Push(path string, replace bool) {
	if ctx.Settings.GetUserConfirmation != nil && ctx.Settings.GetUserConfirmation() {
		return
	}

	ctx.ChangeRoute(path, replace)

	dom.GetWindow().DispatchEvent(js.New("Event", ChangeRouteEvent))
}

// PushDynamic push user to another route with params and queries
func (i RouteInfo) PushDynamic(name string, params, queries map[string]string, replace bool) {
	i.Ctx.PushDynamic(name, params, queries, replace)
}

// PushDynamic push user to another route with params and queries
func (ctx *Ctx) PushDynamic(name string, params, queries map[string]string, replace bool) {
	ctx.Push(ctx.fillPath(name, params, queries), replace)
}

func (ctx *Ctx) fillPath(name string, params, queries map[string]string) string {
	route := ctx.getRoute(name)
	if route.Name == "" {
		return ""
	}

	path := route.Path

	for x := 0; x < ctx.Settings.MaxRouteParams; x++ {
		p1, name, p2 := splitPath(path)
		if len(name) == 0 {
			var queriesString string
			if queries != nil {
				queriesString = "?"
				for key, value := range queries {
					queriesString = queriesString + key + "=" + value + "&"
				}
				queriesString = strings.TrimSuffix(queriesString, "&") // remove last "&"
			}

			return path + queriesString
		}

		path = fmt.Sprintf("%s%s%s", p1, params[name], p2)
	}

	ctx.This.c.WarnError(fmt.Errorf("invalid path"))
	return path
}

func (ctx *Ctx) getRoute(name string) Route {
	for _, r := range ctx.Routes {
		if r.Name == name {
			return r
		}
	}

	ctx.This.c.WarnError(fmt.Errorf("undefined route: %s", name))
	return Route{}
}

func (ctx Ctx) link(path string, push func(gas.Object), e gas.External) *gas.Element {
	if e.Attrs == nil {
		e.Attrs = make(map[string]string)
	}
	
	e.Attrs["href"] = ctx.Settings.BaseName + path

	return gas.NE(
		&gas.E{
			Tag: "a",
			Attrs: e.Attrs,
			Handlers: map[string]gas.Handler {
				"click":    beforePush(push),
				"keyup.13": beforePush(push),
				"keyup.32": beforePush(push),
			},
		},
		e.Body...)
}
func beforePush(push func(gas.Object)) func(gas.Object) {
	return func(event gas.Object) {
		push(event)
		event.Call("preventDefault")
	}
}

// Link create link to route
func (i RouteInfo) Link(to string, replace bool, e gas.External) *gas.Element {
	return i.Ctx.Link(to, replace, e)
}

// Link create link to route
func (ctx *Ctx) Link(to string, replace bool, e gas.External) *gas.Element {
	return ctx.link(
		to,
		func(e gas.Object) {
			ctx.Push(to, replace)
		},
		e)
}

// LinkWithParams create link to route with queries and params
func (i RouteInfo) LinkWithParams(name string, params, queries map[string]string, replace bool, e gas.External) *gas.Element {
	return i.Ctx.LinkWithParams(name, params, queries, replace, e)
}

//LinkWithParams create link to route with queries and params
func (ctx *Ctx) LinkWithParams(name string, params, queries map[string]string, replace bool, e gas.External) *gas.Element {
	return ctx.link(
		ctx.fillPath(name, params, queries),
		func(e gas.Object) {
			ctx.PushDynamic(name, params, queries, replace)
		},
		e)
}
