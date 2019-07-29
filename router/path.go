package router

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gascore/dom"
	"github.com/gascore/dom/js"
	sjs "syscall/js"
)

func (ctx *Ctx) getRoute(name string) Route {
	for _, r := range ctx.Routes {
		if r.Name == name {
			return r
		}
	}

	ctx.This.c.WarnError(fmt.Errorf("undefined route: %s", name))
	return Route{}
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
	if len(ctx.renderedPaths[route.Path]) != 0 {
		path = ctx.renderedPaths[route.Path]
	} else {
		path = renderPath(route.Path, ctx)
		ctx.renderedPaths[route.Path] = path
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