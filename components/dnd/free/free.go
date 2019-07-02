package dndfree

import (
	"fmt"

	"github.com/frankenbeanies/uuid4"

	"github.com/gascore/gas/web"

	"github.com/gascore/dom"
	"github.com/gascore/dom/js"
	"github.com/gascore/gas"
)

// Config dnd-free config structure
type Config struct {
	Tag string

	XDisabled bool // only Y
	YDisabled bool // only X

	ifXDisabled int // 0 if disabled, 1 if enabled
	ifYDisabled int // 0 if disabled, 1 if enabled

	Boundary string // boundary element class
	Handle   string // handle element class

	Class string

	OnMove  func(event gas.Object, x, y int) error
	OnStart func(event gas.Object, x, y int) (block bool, err error)
	OnEnd   func(event gas.Object, x, y int) (reset bool, err error)
}

func (config *Config) normalize() {
	if config.Tag == "" {
		config.Tag = "div"
	}

	if config.Class == "" {
		config.Class = "dnd-free"
	}

	if config.XDisabled {
		config.ifXDisabled = 0
	} else {
		config.ifXDisabled = 1
	}

	if config.YDisabled {
		config.ifYDisabled = 0
	} else {
		config.ifYDisabled = 1
	}
}

// DNDFree free draggable component
func DNDFree(config *Config, e gas.External) *gas.E {
	config.normalize()

	if config.XDisabled && config.YDisabled {
		dom.ConsoleError("x and y are disabled: element is static")
		return nil
	}

	childUUID := uuid4.New().String()

	root := &dndEl{
		e:         e,
		config:    config,
		childUUID: childUUID,
	}

	c := &gas.C{
		Root: root,
		Element: &gas.E{
			Tag: config.Tag,
			Attrs: map[string]string{
				"class": config.Class + "-wrap",
			},
		},
		Hooks: gas.Hooks{
			Mounted: func() error {
				var _boundary *dom.Element
				if config.Boundary != "" {
					_boundary = dom.Doc.QuerySelector("." + config.Boundary)
					if _boundary == nil {
						dom.ConsoleError("boundary is undefined")
					}
				}

				moveEvent := event(func(event dom.Event) {
					if !root.isActive {
						return
					}

					event.PreventDefault()

					var x, y int
					if event.Type() == "touchmove" {
						t := event.JSValue().Get("touches").Get("0")
						x = t.Get("clientX").Int()
						y = t.Get("clientY").Int()
					} else {
						x = event.JSValue().Get("clientX").Int()
						y = event.JSValue().Get("clientY").Int()
					}

					if _boundary != nil {
						rect := _boundary.JSValue().Call("getBoundingClientRect")

						var (
							left   = rect.Get("left").Int()
							top    = rect.Get("top").Int()
							bottom = rect.Get("bottom").Int()
							right  = rect.Get("right").Int()

							cursorOffsetLeft   = root.cursorOffsetLeft
							cursorOffsetTop    = root.cursorOffsetTop
							cursorOffsetRight  = root.cursorOffsetRight
							cursorOffsetBottom = root.cursorOffsetBottom
						)

						if (x - cursorOffsetLeft) <= left {
							x = left + cursorOffsetLeft
						} else if (x + cursorOffsetRight) >= right {
							x = right - cursorOffsetRight
						}

						if (y - cursorOffsetTop) <= top {
							y = top + cursorOffsetTop
						} else if (y + cursorOffsetBottom) >= bottom {
							y = bottom - cursorOffsetBottom
						}
					}

					x = (x - root.initialX) * config.ifXDisabled
					y = (y - root.initialY) * config.ifYDisabled

					root.offsetX = x
					root.offsetY = y

					go root.c.Update()

					if config.OnMove != nil {
						err := config.OnMove(web.ToUniteObject(event), x, y)
						if err != nil {
							root.c.ConsoleError(err.Error())
						}
					}
				})

				startEvent := event(func(event dom.Event) {
					if config.Handle == "" {
						_target := event.Target()
						if _target.GetAttribute("data-i").String() != root.childUUID && !root.c.Element.BEElement().(*dom.Element).Contains(_target) {
							return
						}
					} else if !event.Target().ClassList().Contains(config.Handle) {
						return
					}

					var clientX, clientY int
					if event.Type() == "touchstart" {
						t := event.JSValue().Get("touches").Get("0")
						clientX = t.Get("clientX").Int()
						clientY = t.Get("clientY").Int()
					} else {
						clientX = event.JSValue().Get("clientX").Int()
						clientY = event.JSValue().Get("clientY").Int()
					}

					x := clientX - root.offsetX
					y := clientY - root.offsetY

					if config.OnStart != nil {
						block, err := config.OnStart(web.ToUniteObject(event), x, y)
						if err != nil {
							root.c.WarnError(err)
							return
						}
						if block {
							return
						}
					}

					root.initialX = x
					root.initialY = y
					root.isActive = true

					if _boundary != nil {
						rect := dom.Doc.QuerySelector("[data-i='" + root.childUUID + "']").JSValue().Call("getBoundingClientRect")

						root.cursorOffsetLeft = clientX - rect.Get("left").Int()
						root.cursorOffsetTop = clientY - rect.Get("top").Int()
						root.cursorOffsetRight = rect.Get("right").Int() - clientX
						root.cursorOffsetBottom = rect.Get("bottom").Int() - clientY
					}

					go root.c.Update()
				})

				endEvent := event(func(event dom.Event) {
					if !root.isActive {
						return
					}

					if config.OnEnd != nil {
						reset, err := config.OnEnd(web.ToUniteObject(event), root.offsetX, root.offsetY)
						if err != nil {
							root.c.WarnError(err)
							return
						}

						if reset {
							root.initialX = 0
							root.initialY = 0
							root.offsetX = 0
							root.offsetY = 0
							root.isActive = false
							go root.c.Update()
							return
						}
					}

					root.initialX = root.offsetX
					root.initialY = root.offsetY
					root.isActive = false

					go root.c.Update()
				})

				addEvent(dom.Doc, "mousemove", moveEvent)
				addEvent(dom.Doc, "mousedown", startEvent)
				addEvent(dom.Doc, "mouseup", endEvent)

				root.moveEvent = moveEvent
				root.startEvent = startEvent
				root.endEvent = endEvent

				return nil
			},
			BeforeDestroy: func() error {
				removeEvent(dom.Doc, "mousemove", root.moveEvent)
				removeEvent(dom.Doc, "mousedown", root.startEvent)
				removeEvent(dom.Doc, "mouseup", root.endEvent)

				return nil
			},
		},
	}
	root.c = c

	return c.Init()
}

type dndEl struct {
	c         *gas.C
	e         gas.External
	config    *Config
	childUUID string

	initialX int
	initialY int

	offsetX int
	offsetY int

	cursorOffsetLeft   int
	cursorOffsetTop    int
	cursorOffsetRight  int
	cursorOffsetBottom int

	isActive bool

	startEvent js.Func
	endEvent   js.Func
	moveEvent  js.Func
}

func (root *dndEl) Render() []interface{} {
	subRoot := &dndSubEl{
		e: root.e,
	}

	c := &gas.C{
		Root:               subRoot,
		ElementIsImportant: true,
		Element: &gas.E{
			UUID: root.childUUID,
			Binds: map[string]gas.Bind{
				"style": func() string {
					return fmt.Sprintf("transform: translate3d(%dpx, %dpx, 0px)", root.offsetX, root.offsetY)
				},
				"class": func() string {
					var isActiveClass string
					if root.isActive {
						isActiveClass = root.config.Class + "-active"
					}
					return root.config.Class + " " + isActiveClass
				},
			},
		},
		Hooks: gas.Hooks{
			Mounted: func() error {
				_el := subRoot.c.Element.BEElement().(*dom.Element)

				addEvent(_el, "touchstart", root.startEvent)
				addEvent(_el, "touchend", root.endEvent)
				addEvent(_el, "touchcancel", root.endEvent)
				addEvent(_el, "touchmove", root.moveEvent)

				return nil
			},
		},
	}
	subRoot.c = c

	return gas.CL(c.Init())
}

type dndSubEl struct {
	c *gas.C
	e gas.External
}

func (root *dndSubEl) Render() []interface{} {
	return root.e.Body
}

func addEvent(e dom.Node, typ string, h js.Func) {
	e.JSValue().Call("addEventListener", typ, h)
}

func removeEvent(e dom.Node, typ string, h js.Func) {
	e.JSValue().Call("removeEventListener", typ, h)
}

func event(f func(event dom.Event)) js.Func {
	return js.NewEventCallback(func(v js.Value) {
		f(dom.ConvertEvent(v))
	})
}
