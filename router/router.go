package router

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gascore/dom"
	"github.com/gascore/dom/js"
	"github.com/gascore/gas"
	sjs "syscall/js"
)

// ChangeRouteEvent name for custom event
const ChangeRouteEvent = "changeroute"

// renderedPaths dummy cache for rendered paths 
var renderedPaths = make(map[string]string)

// Route information about route
type Route struct {
	Name      string
	Element   func(info RouteInfo) *gas.Element
	Exact     bool
	Sensitive bool

	Redirect        string
	RedirectParams  map[string]string // Route.RP != nil || Route.RQ != nil => Route.Redirect is route name
	RedirectQueries map[string]string

	Path    string
	ArrPath []string
}

// Ctx router context
type Ctx struct {
	Routes   []Route
	Settings Settings
	This     *routerComponent
}

// Settings router settings
type Settings struct {
	BaseName string

	HashMode   bool
	HashSuffix string // "!", "/" for "#!", "#/"

	GetUserConfirmation func() bool
	ForceRefresh        bool

	Redirect *gas.Element
	NotFound *gas.Element

	MaxRouteParams int
}

// Init initialize router ctx
func (ctx *Ctx) Init() *Ctx {
	if ctx.Settings.NotFound == nil {
		ctx.Settings.NotFound = gas.NE(&gas.E{}, "404. Page not found")
	}

	if ctx.Settings.Redirect == nil {
		ctx.Settings.Redirect = gas.NE(&gas.E{}, "Redirecting")
	}

	if ctx.Settings.HashMode {
		ctx.Settings.BaseName = "#" + ctx.Settings.HashSuffix + ctx.Settings.BaseName
	}

	if ctx.Settings.MaxRouteParams == 0 {
		ctx.Settings.MaxRouteParams = 64
	}

	for _, route := range ctx.Routes {
		if len(route.Redirect) == 0 {
			continue
		}

		if route.RedirectParams == nil {
			route.RedirectParams = make(map[string]string)
		}

		if route.RedirectQueries == nil {
			route.RedirectQueries = make(map[string]string)
		}
	}

	return ctx
}

// GetRouter return gas router element
func (ctx *Ctx) GetRouter() *gas.Element {
	root := &routerComponent{
		ctx: ctx,
	}

	c := &gas.C{
		Root: root,
		Hooks: gas.Hooks{
			Mounted: func() error {
				root.updateEvent = event(func(e dom.Event) {
					root.c.Update() 
				})

				windowAddEventListener("popstate", root.updateEvent)
				windowAddEventListener(ChangeRouteEvent, root.updateEvent)

				return nil
			},
			BeforeDestroy: func() error {
				windowRemoveEventListener("popstate", root.updateEvent)
				windowRemoveEventListener(ChangeRouteEvent, root.updateEvent)

				return nil
			},
		},
	}
	root.c = c

	return c.Init()
}

type routerComponent struct {
	c *gas.C
	ctx *Ctx

	lastItem    *gas.Element
	lastRoute   string
	updateEvent js.Func
}

func(root *routerComponent) Render() []interface{} {
	ctx := root.ctx
	if !strings.HasPrefix(ctx.getPath(), ctx.Settings.BaseName) {
		ctx.ChangeRoute("/", true)
	}

	currentPath := strings.TrimPrefix(ctx.getPath(), ctx.Settings.BaseName)
	if currentPath == "" {
		currentPath = "/"
	}

	ctx.This = root

	return gas.CL(
		gas.NE(
			&gas.E{
				Attrs: map[string]string{
					"data-path": currentPath,
					"id":        "gas-router_route-wraper",
				},
			},
			ctx.findRoute(currentPath),
		),
	)
}

// ChangeRoute change current route
func (i RouteInfo) ChangeRoute(path string, replace bool) {
	i.Ctx.ChangeRoute(path, replace)
}

// ChangeRoute change current route
func (ctx *Ctx) ChangeRoute(path string, replace bool) {
	path = ctx.Settings.BaseName + path

	if ctx.Settings.ForceRefresh {
		dom.GetWindow().JSValue().Set(
			"location",
			dom.GetWindow().GetLocation().Get("origin").String()+path)
		return
	}

	if replace {
		dom.GetWindow().GetHistory().Call("replaceState", "", "", path)
	} else {
		dom.GetWindow().GetHistory().Call("pushState", "", "", path)

		c  := ctx.This.c
		el := c.Element

		_el, ok := el.BEElement().(*dom.Element)
		if !ok || _el == nil {
			c.ConsoleError("invalid ctx.This")
			return
		}

		if len(el.RChildes) == 0 {
			c.ConsoleError("invalid ctx.This RChildes")
			return
		}

		routeWrap, ok := el.RChildes[0].(*gas.E)
		if !ok {
			c.ConsoleError("invalid gas-router route wrapper type")
			return
		}

		routeWrap.Attrs["data-path"] = path

		elChildes := _el.ChildNodes()
		if len(elChildes) == 0 {
			c.ConsoleError("invalid _el ChildNodes")
			return
		}

		elChildes[0].SetAttribute("data-path", path)
	}
}

// ChangeRouteDynamic change current route with params and queries
func (i RouteInfo) ChangeRouteDynamic(name string, params, queries map[string]string, replace bool) {
	i.Ctx.ChangeRouteDynamic(name, params, queries, replace)
}

// ChangeRouteDynamic change current route with params and queries
func (ctx *Ctx) ChangeRouteDynamic(name string, params, queries map[string]string, replace bool) {
	ctx.ChangeRoute(ctx.fillPath(name, params, queries), replace)
}

func (ctx *Ctx) findRoute(currentPath string) *gas.Element {
	root := ctx.This
	c    := root.c

	if currentPath == root.lastRoute {
		return root.lastItem
	}

	for _, route := range ctx.Routes {
		routeIsFits, params, queries, err := ctx.matchPath(currentPath, route)
		if err != nil {
			c.ConsoleError("error in router: ", err.Error())
			return nil
		}

		if !routeIsFits {
			continue
		}

		if len(route.Redirect) != 0 {
			if route.RedirectParams == nil && route.RedirectQueries == nil {
				ctx.Push(route.Redirect, true)
			} else {
				ctx.PushDynamic(route.Redirect, route.RedirectParams, route.RedirectQueries, true)
			}

			return ctx.Settings.Redirect
		}

		newEl := route.Element(
			RouteInfo{
				Name: route.Name,
				URL:  currentPath,

				Params:      params,
				QueryParams: queries,

				Route: route,

				Ctx: ctx,
			},
		)

		root.lastRoute = currentPath
		root.lastItem  = newEl

		return newEl
	}

	return ctx.Settings.NotFound
}

func (ctx *Ctx) getPath() string {
	if ctx.Settings.HashMode {
		return dom.GetWindow().GetLocation().Get("hash").String()
	}
	
	return dom.GetWindow().GetLocationPath()
}

func (ctx *Ctx) matchPath(currentPath string, route Route) (bool, map[string]string, map[string]string, error) {
	params, queries := make(map[string]string), make(map[string]string)
	if route.Exact && currentPath == route.Path {
		return true, params, queries, nil
	}

	if strings.HasPrefix(currentPath, route.Path) && !route.Exact {
		return true, params, queries, nil
	}

	// need to cache it
	var path string
	if len(renderedPaths[route.Path]) != 0 {
		path = renderedPaths[route.Path]
	} else {
		path = renderPath(route.Path, ctx)
		renderedPaths[route.Path] = path
	}

	r, err := regexp.Compile(path)
	if err != nil {
		return false, nil, nil, errors.New("invalid path name")
	}

	matches := r.FindStringSubmatch(currentPath)
	if len(matches) <= 1 {
		return false, nil, nil, nil
	}

	names := r.SubexpNames()
	for i, match := range matches {
		if i == 0 {
			continue
		}

		params[names[i]] = match
	}

	splitPath := strings.Split(dom.GetWindow().GetLocation().Get("href").String(), "?")
	if len(splitPath) > 1 { // some.com/wow?foo=bar&some=wow  =>  ["some.com/wow", "foo=bar&some=wow"]
		for _, query := range strings.Split(splitPath[1], "&") {
			if len(query) == 0 {
				continue
			}

			splitQuery := strings.Split(query, "=")
			if len(splitQuery) != 2 {
				ctx.This.c.WarnError(fmt.Errorf("invalid query parametr: %s", query))
				continue
			}

			queries[splitQuery[0]] = splitQuery[1]
		}
	}

	return true, params, queries, nil
}

func renderPath(a string, ctx *Ctx) string {
	path := a
	for x := 0; x < 64; x++ {
		p1, name, p2 := splitPath(path)
		if len(name) == 0 {
			return p1
		}

		val := fmt.Sprintf(`(?P<%s>[^/?=]*)`, name)
		path = p1 + val + p2
	}

	ctx.This.c.ConsoleError(fmt.Sprintf("invalid path: %s", a))
	return a
}

func splitPath(path string) (string, string, string) {
	index := strings.Index(path, ":")
	if index == -1 {
		return path, "", ""
	}

	slashIndex := strings.Index(path[index:], "/")
	if slashIndex == -1 {
		slashIndex = len(path)
	}

	return path[:index], path[index+1 : slashIndex], path[slashIndex:]
}

// SupportHistory return ture if browser support "HTML5 History API"
func SupportHistory() bool {
	return dom.GetWindow().GetHistory().Type().String() != "undefined" &&
		dom.GetWindow().GetHistory().Get("pushState").Type().String() != "undefined" &&
		dom.GetWindow().JSValue().Get("CustomEvent").Type() == sjs.TypeFunction
}

func event(h func(event dom.Event)) js.Func {
	return js.NewEventCallback(func(v js.Value) {
		h(dom.ConvertEvent(v))
	})
}

func windowAddEventListener(eType string, f js.Func) {
	dom.GetWindow().JSValue().Call("addEventListener", eType, f)
}

func windowRemoveEventListener(eType string, f js.Func) {
	dom.GetWindow().JSValue().Call("removeEventListener", eType, f)
}
