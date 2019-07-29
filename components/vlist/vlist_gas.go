package vlist

import (
	"fmt"
	"strconv"

	"github.com/gascore/dom"
	"github.com/gascore/dom/js"
	"github.com/gascore/gas"
)

type Config struct {
	Direction bool // true - vertical, false - horizontal
	directionV, directionTransform string

	ItemsWrapperTag string // ul, table, dl
	Padding         int
	Items           []interface{}

	DefaultItemsSmooth int

	ChildHeight int
	ChildWidth  int
	childSize   int

	Change func(start, end int) error
}

func (config *Config) normalize() {
	if config.Direction {
		config.directionTransform = "translateY"
		config.directionV = "height"
		config.childSize = config.ChildHeight
	} else {
		config.directionTransform = "translateX"
		config.directionV = "width"
		config.childSize = config.ChildWidth
	}

	if config.DefaultItemsSmooth == 0 {
		config.DefaultItemsSmooth = 4
	}

	if config.ItemsWrapperTag == "" {
		config.ItemsWrapperTag = "div"
	}
}

type Renderer func(item interface{}, i, start int) (el *gas.E)

type vlistEl struct {
	c *gas.C

	config *Config
	
	topPadding   int
	items      	 []interface{}
	scrollHeight int
	start        int

	renderer Renderer
}

func GetList(config *Config, renderer Renderer) *gas.E {
	config.normalize()

	if renderer == nil {
		dom.ConsoleError("invalid renderer for vlist")
		return nil
	}

	root := &vlistEl{
		config: config,
		scrollHeight: config.childSize * len(config.Items),
		renderer: renderer,
	}

	c := &gas.C{
		Element: &gas.E{
			Attrs: map[string]string{
				"style": "height: 100%; width: 100%",
			},
		},
		Root: root,
		Hooks: gas.Hooks{
			Mounted: func() error {
				root.onScroll()
				return nil
			},
		},
	}
	root.c = c

	return c.Init()
}

func (root *vlistEl) Render() []interface{} {
	return gas.CL(gas.NE(&gas.E{Tag:"div", Handlers: map[string]gas.Handler{"scroll": func(e gas.Object) {root.onScroll()},},Attrs: map[string]string{"class": "vlist",},},gas.NE(&gas.E{Tag:"div", Binds: map[string]gas.Bind{"style": func() string { return (fmt.Sprintf(`%s: %dpx;`, root.config.directionV, root.scrollHeight))},},Attrs: map[string]string{"class": "vlist-padding",},},),root.genItems(),),)
}

func (root *vlistEl) onScroll() {
	dom.GetWindow().RequestAnimationFrame(func(timeStep js.Value) {
		var err error
		root.items, err = root.calculateItems()
		if err != nil {
			dom.ConsoleError(err.Error())
			return
		}

		go root.c.Update()
	})
}

func (root *vlistEl) calculateItems() ([]interface{}, error) {
	_el := root.c.Element.BEElement().(*dom.Element)
	if _el == nil {
		return []interface{}{}, nil
	}

	var scrollTop, offsetHeight int
	if root.config.Direction {
		scrollTop = _el.ChildNodes()[0].ScrollTop()
		offsetHeight = _el.OffsetHeight()
	} else {
		scrollTop = _el.ChildNodes()[0].ScrollLeft()
		offsetHeight = _el.OffsetWidth()
	}

	endRaw, err := strconv.Atoi(fmt.Sprintf("%.0f", float64(offsetHeight)/float64(root.config.childSize)))
	if err != nil {
		return []interface{}{}, err
	}

	start := scrollTop / root.config.childSize
	end := start + endRaw

	root.start = start
	root.topPadding = start*root.config.childSize

	if root.config.Change != nil {
		err = root.config.Change(start, end)
		if err != nil {
			return []interface{}{}, err
		}
	}

	return getSlice(root.config.Items, start, end), nil
}

func (root *vlistEl) genItems() *gas.E {
	if len(root.items) == 0 {
		var err error
		root.items, err = root.calculateItems()
		if err != nil {
			dom.ConsoleError(err.Error())
			return nil
		}
	}

	var items []interface{}
	for i, item := range root.items {
		items = append(items, root.renderer(item, i, root.start))
	}

	return gas.NE(
		&gas.E{
			Tag: root.config.ItemsWrapperTag,
			Binds: map[string]gas.Bind{
				"style": func() string {
					var dFlex string
					if !root.config.Direction {
						dFlex = "display: flex;"
					}
					return fmt.Sprintf(`transform: %s(%dpx);%s`, root.config.directionTransform, root.topPadding, dFlex)
				},
			},
			Attrs: map[string]string{
				"class": "vlist-content",
			},
		},
		items,
	)
}

func getSlice(arr []interface{}, start, end int) []interface{} {
	if start > len(arr) {
		return []interface{}{}
	}

	if end > len(arr) {
		end = len(arr)
	}

	return arr[start:end]
}