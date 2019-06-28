package layout

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gascore/dom"
	"github.com/gascore/dom/js"
	"github.com/gascore/gas"
	sjs "syscall/js"
)

// Config layout config structure
type Config struct {
	DragInterval int

	LayoutClass string
	GutterClass string
	GutterSize  int

	Sizes []Size

	Type bool // true - horizontal, false - vertical

	OnStart Event
	OnStop  Event // runs before Element recreating
	OnMove  MoveEvent

	byGuttersOffset float64
	allGuttersSize  int

	typeString     string
	orientation    string
	orientationB   string
	subOrientation string
	clientAxis     string
	positionEnd    string
}

func (config *Config) normalize() {
	if config.DragInterval == 0 {
		config.DragInterval = 1
	}

	if config.GutterSize == 0 {
		config.GutterSize = 2
	}

	if config.Type {
		config.typeString = "horizontal"
		config.orientation = "width"
		config.orientationB = "Width"
		config.subOrientation = "height"
		config.clientAxis = "clientX"
		config.positionEnd = "right"
	} else {
		config.typeString = "vertical"
		config.orientation = "height"
		config.orientationB = "Height"
		config.subOrientation = "width"
		config.clientAxis = "clientY"
		config.positionEnd = "bottom"
	}

	if config.LayoutClass == "" {
		config.LayoutClass = "layout"
	}

	if config.GutterClass == "" {
		config.GutterClass = "gutter"
	}
}

type Event func(first, second Element, _gutter *dom.Element) (stopIt bool, err error)
type MoveEvent func(first, second Element, _gutter *dom.Element, offset float64) (stopIt bool, err error)

// Size layout item size info
type Size struct {
	Min     float64
	Max     float64
	Start   float64
	current float64
}

type Element struct {
	E     *gas.E
	Index int
}

type layoutEl struct {
	c *gas.C

	elements []Element

	sizes []Size
	config *Config
}

func (root *layoutEl) Render() []interface{} {
	var childes []interface{}

	config := root.config

	for i, child := range root.elements {
		childes = append(childes, gas.NE(
			&gas.E{
				Attrs: map[string]string{
					"class":  config.LayoutClass + "-item",
					"style":  fmt.Sprintf("%s: calc(%f%s - %fpx); %s: 100%s;", config.orientation, root.sizes[child.Index].current, "%", config.byGuttersOffset, config.subOrientation, "%"),
					"data-i": fmt.Sprintf("%d", i),
				},
			},
			child.E,
		))
		if i != len(root.elements)-1 {
			childes = append(childes, gutter(root, config, child, root.elements[i+1]))
		}
	}

	return childes
}

func (root *layoutEl) GetSizes() []Size {
	return root.sizes
}

func (root *layoutEl) SetSizes(newSizes []Size) {
	root.sizes = newSizes
	go root.c.Update()
}

// Layout return resizable layout
func Layout(config *Config, e gas.External) *gas.Element {
	e.Body = gas.RemoveStrings(e.Body)

	if len(e.Body) != len(config.Sizes) {
		dom.ConsoleError("not enough Element sizes")
		return nil
	}

	config.normalize()

	var sizesSum float64 // check sizes sum == 100 && make Size.Start valid
	for _, size := range config.Sizes {
		if size.Start > size.Max {
			size.Start = size.Max
		} else if size.Min > size.Start {
			size.Start = size.Min
		}
		sizesSum += size.Start
	}

	if sizesSum != 100 {
		// because i don't want to create an implicit state change
		dom.ConsoleError("invalid sizes: size.Start sum != 100")
		return nil
	}

	var elements []Element
	var sizes []Size
	for i, child := range e.Body {
		childE, ok := child.(*gas.E)
		if !ok {
			dom.ConsoleError(fmt.Sprintf("invalid child in layout - child is not element: '%T'", child))
			return nil
		}

		if childE.Attrs == nil {
			childE.Attrs = make(map[string]string)
		}

		size := config.Sizes[i]
		size.current = size.Start

		sizes = append(sizes, size)
		elements = append(elements, Element{E: childE, Index: i})
	}

	config.allGuttersSize = (len(elements) - 1) * config.GutterSize
	config.byGuttersOffset = float64(config.allGuttersSize) / float64(len(elements))

	root := &layoutEl {
		sizes: sizes,
		config: config,
		elements: elements,
	}

	c := &gas.C {
		Root: root,
		Element: &gas.E{
			Attrs: map[string]string{
				"class": fmt.Sprintf("%s %s-%s", config.LayoutClass, config.LayoutClass, config.typeString),
			},
		},
	}
	root.c = c

	return c.Init()
}

type sizesFubInterface interface{
	GetSizes()[]Size
	SetSizes([]Size)
}

type gutterEl struct {
	c *gas.C

	dragOffset float64
	dragging   bool

	startEvent, moveEvent, stopEvent js.Func
}

func gutter(sizesFub sizesFubInterface, config *Config, first, second Element) *gas.Element {
	var cursorType string
	if config.Type {
		cursorType = "ew-resize"
	} else {
		cursorType = "row-resize"
	}

	root := &gutterEl{
	}

	c := &gas.C{
		Root: root,
		Element: &gas.E{
			Tag: "div",
			Attrs: map[string]string {
				"class": fmt.Sprintf("%s %s-%s", config.GutterClass, config.GutterClass, config.typeString),
				"style": fmt.Sprintf("cursor: %s; %s: %dpx", cursorType, config.orientation, config.GutterSize),
			},
		},
		Hooks: gas.Hooks{
			Mounted: func() error {
				_el := root.c.Element.BEElement().(*dom.Element)

				computedStyles := sjs.Global().Call("getComputedStyle", _el.JSValue())
				var parentSize string
				if config.Type {
					parentSize = fmt.Sprintf("%dpx", _el.ParentElement().ClientHeight() - parseP(computedStyles.Get("paddingTop")) - parseP(computedStyles.Get("paddingBottom")))
				} else {
					parentSize = "100%"
				}
				_el.Style().Set(config.subOrientation, parentSize)

				moveEvent := event(func(event dom.Event) error {
					if !root.dragging {
						return nil
					}

					event.PreventDefault()

					/*_el = event.Target()
					if _el.ParentElement() == nil {
						dom.ConsoleLog(_el.JSValue())
						log.Fatal("WTF?!?!??!?!")
					}*/

					var start float64
					if config.Type {
						start = _el.GetBoundingClientRectRaw().Get("left").Float()
					} else {
						start = _el.GetBoundingClientRectRaw().Get("top").Float()
					}

					offset := getMousePosition(config.clientAxis, event) - start + float64(config.GutterSize)
					if offset == 0 {
						return nil
					}

					if config.OnMove != nil {
						stopIt, err := config.OnMove(first, second, _el, offset)
						if err != nil {
							return err
						}

						if stopIt {
							return nil
						}
					}

					sizes := sizesFub.GetSizes()
					var newFirst, newSecond float64
					if offset < 0 {
						newFirst, newSecond = getSizes(-offset, _el.ParentElement(), sizes[first.Index], sizes[second.Index], config)
					} else {
						newSecond, newFirst = getSizes(offset, _el.ParentElement(), sizes[second.Index], sizes[first.Index], config)
					}

					sizes[first.Index].current = newFirst
					sizes[second.Index].current = newSecond

					sizesFub.SetSizes(sizes)

					return nil
				})

				stopEvent := event(func(event dom.Event) error {
					if !root.dragging {
						return nil
					}

					_el = root.c.Element.BEElement().(*dom.Element)
					_first := first.E.BEElement().(*dom.Element)
					_second := second.E.BEElement().(*dom.Element)

					if config.OnStop != nil {
						stopIt, err := config.OnStop(first, second, _el)
						if err != nil {
							return err
						}

						if stopIt {
							return nil
						}
					}

					removeEvent(_el, "touchend", root.stopEvent)
					removeEvent(_el, "touchcancel", root.stopEvent)
					removeEvent(_el, "touchmove", root.moveEvent)

					removeEvent(dom.Doc, "mouseup", root.stopEvent)
					removeEvent(dom.Doc, "mousemove", root.moveEvent)

					for _, _x := range []*dom.Element{_first, _second} {
						_x.Style().Set("userSelect", "")
						_x.Style().Set("webkitUserSelect", "")
						_x.Style().Set("MozUserSelect", "")
					}

					root.dragging = false

					return nil
				})

				startEvent := event(func(event dom.Event) error {
					if root.dragging {
						return nil
					}

					_el := event.Target()
					_first := first.E.BEElement().(*dom.Element)
					_second := second.E.BEElement().(*dom.Element)

					if config.OnStart != nil {
						stopIt, err := config.OnStart(first, second, _el)
						if err != nil {
							return err
						}

						if stopIt {
							return nil
						}
					}

					addEvent(_el, "touchend", root.stopEvent)
					addEvent(_el, "touchcancel", root.stopEvent)
					addEvent(_el, "touchmove", root.moveEvent)

					addEvent(dom.Doc, "mouseup", root.stopEvent)
					addEvent(dom.Doc, "mousemove", root.moveEvent)

					for _, _x := range []*dom.Element{_first, _second} {
						_x.Style().Set("userSelect", "none")
						_x.Style().Set("webkitUserSelect", "none")
						_x.Style().Set("MozUserSelect", "none")
					}

					_el.ClassList().Add(config.GutterClass + "-focus")

					root.dragOffset = getMousePosition(config.clientAxis, event) - _first.GetBoundingClientRectRaw().Get(config.positionEnd).Float()
					root.dragging = true

					return nil
				})

				addEvent(_el, "mousedown", startEvent)
				addEvent(_el, "touchstart", startEvent)

				addEvent(_el, "mouseup", stopEvent)
				addEvent(_el, "touchend", stopEvent)
				addEvent(_el, "touchcancel", stopEvent)

				addEvent(_el, "mousemove", moveEvent)
				addEvent(_el, "touchmove", moveEvent)

				root.moveEvent  = moveEvent
				root.startEvent = startEvent
				root.stopEvent  = stopEvent

				return nil
			},
			BeforeDestroy: func() error {
				_el := root.c.Element.BEElement().(*dom.Element)
				if _el == nil {
					return nil
				}

				removeEvent(_el, "mousedown", root.startEvent)
				removeEvent(_el, "touchstart", root.startEvent)

				return nil
			},
		},
	}
	root.c = c

	return c.Init()
}

func (root *gutterEl) Render() []interface{} {
	return gas.CL()
}

func getSizes(offset float64, _parent *dom.Element, first, second Size, config *Config) (float64, float64) {
	offsetOri := "offset" + config.orientationB
	layoutSize := _parent.JSValue().Get(offsetOri).Float() - float64(config.allGuttersSize)
	theirSizeP := first.current + second.current
	offsetP := (offset * 100) / layoutSize

	if first.current-offsetP < first.Min {
		makeSecond := theirSizeP - first.Min
		if makeSecond > second.Max {
			return theirSizeP - second.Max, second.Max
		}

		return first.Min, makeSecond
	}

	if second.current+offsetP > second.Max {
		if second.Max > theirSizeP || second.Max > theirSizeP-first.Min {
			return first.Min, theirSizeP - first.Min
		}

		return theirSizeP - second.Max, second.Max
	}

	return first.current - offsetP, theirSizeP - (first.current - offsetP)
}

func getMousePosition(clientAxis string, event dom.Event) float64 {
	if notJsNull(event.JSValue().Get("touches")) {
		return event.JSValue().Get("touches").Get("0").Get(clientAxis).Float()
	}

	return event.JSValue().Get(clientAxis).Float()
}

func event(f func(event dom.Event) error) js.Func {
	return js.NewEventCallback(func(v js.Value) {
		err := f(dom.ConvertEvent(v))
		if err != nil {
			dom.ConsoleError(err.Error())
			return
		}
	})
}

func addEvent(e dom.Node, typ string, h js.Func) {
	e.JSValue().Call("addEventListener", typ, h)
}

func removeEvent(e dom.Node, typ string, h js.Func) {
	e.JSValue().Call("removeEventListener", typ, h)
}

func notJsNull(e sjs.Value) bool {
	return e.Type() != sjs.TypeUndefined && e.Type() != sjs.TypeNull
}

func parseP(a sjs.Value) int {
	b, _ := strconv.Atoi(strings.TrimSuffix(a.String(), "px"))
	return b
}
