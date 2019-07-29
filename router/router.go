package router

import (
	"fmt"
	"strings"

	"github.com/gascore/dom"
	"github.com/gascore/dom/js"
	"github.com/gascore/gas"
)

// ChangeRouteEvent name for custom event
const ChangeRouteEvent = "changeroute"

// Route information about route
type Route struct {
	Name string
	Path string

	Element func(info *RouteInfo) *gas.Element

	Exact     bool
	Sensitive bool

	Redirect string

	RedirectName    string // redirecting to route name
	RedirectParams  map[string]string // pararms for rederecting to route
	RedirectQueries map[string]string // queries for rederecting to route

	Before, After Middleware

	Childes []Route
}

// Ctx router context
type Ctx struct {
	Routes   []Route
	Settings Settings
	This     *routerComponent

	Before, After func(to, from *RouteInfo) error

	renderedPaths map[string]string
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

// RouteInfo info about current route for router
type RouteInfo struct {
	Name string
	URL  string

	Params      map[string]string // /links/:foo => {"foo": "bar"}
	QueryParams map[string]string // /links?foo=bar => {"foo": "bar"}

	Route Route

	Ctx *Ctx
}

type MiddlewareInfo struct {
	To, From *RouteInfo

	Change Change
	ChangeDynamic ChangeDynamic
}

type Middleware func(info *MiddlewareInfo) (stop bool, err error)

type Change func(path string, replace bool)

type ChangeDynamic func(name string, params, queries map[string]string, replace bool)

// Init initialize router ctx
func (ctx *Ctx) Init() {
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

	ctx.renderedPaths = make(map[string]string)

	var newRoutes []Route
	for _, route := range ctx.Routes {
		if len(route.RedirectName) != 0 {
			if route.RedirectParams == nil {
				route.RedirectParams = make(map[string]string)
			}

			if route.RedirectQueries == nil {
				route.RedirectQueries = make(map[string]string)
			}
		}

		newRoutes = append(newRoutes, decomposeRouteChildes(route)...)
	}
	ctx.Routes = newRoutes
}

func decomposeRouteChildes(route Route) []Route {
	var newRoutes []Route
	for _, c := range route.Childes {
		if route.Before != nil {
			if c.Before == nil {
				c.Before = route.Before
			} else {
				cBefore := c.Before
				c.Before = func(info *MiddlewareInfo) (bool, error) {
					stop, err := route.Before(info)
					if err != nil {
						return stop, err
					}

					if stop {
						return stop, nil
					}

					stop, err = cBefore(info)
					if err != nil {
						return stop, err
					}

					return stop, nil
				}
			}
		}

		if route.After != nil {
			if c.After == nil {
				c.After = route.After
			} else {
				cAfter := c.After
				c.After = func(info *MiddlewareInfo) (bool, error) {
					stop, err := route.After(info)
					if err != nil {
						return stop, err
					}

					if stop {
						return stop, nil
					}

					stop, err = cAfter(info)
					if err != nil {
						return stop, err
					}

					return stop, nil
				}
			}
		}

		c.Path = route.Path + c.Path

		if len(c.Childes) != 0 {
			newRoutes = append(newRoutes, decomposeRouteChildes(c)...)
		}

		newRoutes = append(newRoutes, c)
	}

	route.Childes = []Route{}
	newRoutes = append(newRoutes, route)

	return newRoutes
}


// GetRouter return gas router element
func (ctx *Ctx) GetRouter() *gas.Element {
	root := &routerComponent{
		ctx: ctx,
	}

	c := &gas.C{
		NotPointer: true,
		Root:       root,
		Hooks: gas.Hooks{
			Mounted: func() error {
				root.updateEvent = event(func(e dom.Event) {
					from := root.lastRouteInfo

					root.c.Update()

					to := root.lastRouteInfo
					if to.Route.After != nil {
						_, err := to.Route.After(&MiddlewareInfo{
							To: to, 
							From: from, 
							Change: ctx.CustomPush, 
							ChangeDynamic: ctx.CustomPushDynamic,
						})
						if err != nil {
							root.c.ConsoleError(err.Error())
							return
						}
					}

					if ctx.After != nil {
						err := ctx.After(to, from)
						if err != nil {
							root.c.ConsoleError(err.Error())
							return
						}
					}
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
	c   *gas.C
	ctx *Ctx

	lastRouteInfo *RouteInfo
	lastItem      *gas.Element
	lastRoute     string
	updateEvent   js.Func
}

func (root *routerComponent) Render() []interface{} {
	if !strings.HasPrefix(root.ctx.getPath(), root.ctx.Settings.BaseName) {
		root.ctx.ChangeRoute("/", true)
	}

	currentPath := strings.TrimPrefix(root.ctx.getPath(), root.ctx.Settings.BaseName)
	if currentPath == "" {
		currentPath = "/"
	}

	return gas.CL(
		gas.NE(
			&gas.E{
				Attrs: map[string]string{
					"data-path": currentPath,
					"id":        "gas-router_route-wraper",
				},
			},
			root.findRoute(currentPath),
		),
	)
}

func (root *routerComponent) findRoute(currentPath string) *gas.Element {
	ctx := root.ctx
	if currentPath == root.lastRoute {
		return root.lastItem
	}

	for _, route := range ctx.Routes {
		routeIsFits, params, queries, err := ctx.matchPath(currentPath, route)
		if err != nil {
			root.c.ConsoleError(fmt.Sprintf("error in router: %s", err.Error()))
			return nil
		}

		if !routeIsFits {
			continue
		}

		to := &RouteInfo{
			Name: route.Name,
			URL:  currentPath,

			Params:      params,
			QueryParams: queries,

			Route: route,

			Ctx: ctx,
		}

		if ctx.Before != nil {
			err := ctx.Before(to, root.lastRouteInfo)
			if err != nil {
				root.c.ConsoleError(err.Error())
				return root.lastItem // don't update route
			}
		}

		if route.Before != nil {
			var newPath string
			newReplace := true

			stop, err := route.Before(&MiddlewareInfo{
				To: to, 
				From: root.lastRouteInfo, 
				Change: func(path string, replace bool) {
					newPath = path
					newReplace = replace
				}, 
				ChangeDynamic: func(name string, params, queries map[string]string, replace bool) {
					newPath = ctx.fillPath(name, params, queries)
					newReplace = replace
				},
			})
			if err != nil {
				root.c.ConsoleError(err.Error())
			}

			if len(newPath) != 0 && stop {
				ctx.ChangeRoute(newPath, newReplace)
				return root.findRoute(newPath)
			}
		}

		if len(route.Redirect) != 0 {
			ctx.ChangeRoute(route.Redirect, true)
			return root.findRoute(route.Redirect)
		}

		if len(route.RedirectName) != 0 {
			path := ctx.fillPath(route.RedirectName, route.RedirectParams, route.RedirectQueries)
			ctx.ChangeRoute(path, true)
			return root.findRoute(path)
		}

		root.lastRouteInfo = to
		root.lastRoute = currentPath
		root.lastItem = route.Element(to)

		return root.lastItem
	}

	return ctx.Settings.NotFound
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
	}
}

// ChangeRouteDynamic change current route with params and queries
func (ctx *Ctx) ChangeRouteDynamic(name string, params, queries map[string]string, replace bool) {
	ctx.ChangeRoute(ctx.fillPath(name, params, queries), replace)
}
