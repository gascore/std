// Code generated by gasx. DO NOT EDIT.
// source: vlist.gas

package vlist

import (
	"fmt"
	"strconv"

	"github.com/gascore/dom"
	"github.com/gascore/dom/js"
	"github.com/gascore/gas"
)

type Config struct {
	Direction                      bool // true - vertical, false - horizontal
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

type Builder func(e gas.External) *gas.C

func List(config *Config, e gas.External) *gas.C {
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

	if _, ok := e.Templates["item"]; !ok {
		dom.ConsoleError("invalid item template for virtual list")
		return nil
	}

	return gas.NC(
		&gas.C{
			Data: map[string]interface{}{
				"topPadding":   0,
				"items":        []interface{}{},
				"scrollHeight": config.childSize * len(config.Items),
				"start":        0,
			},
			Attrs: map[string]string{
				"style": "height: 100%; width: 100%",
			},
			Hooks: gas.Hooks{
				Mounted: func() error {
					items := this.Method("calculateItems").([]interface{})

					this.SetValue("items", items)

					return nil
				},
			},
			Methods: map[string]gas.Method{
				"onScroll": func(this *gas.Component, values ...interface{}) (interface{}, error) {
					dom.GetWindow().RequestAnimationFrame(func(timeStep js.Value) {
						this.Method("updateItems")
					})

					return nil, nil
				},
				"updateItems": func(this *gas.Component, values ...interface{}) (interface{}, error) {
					items := this.Method("calculateItems").([]interface{})

					this.SetValue("items", items)

					return nil, nil
				},
				"calculateItems": func(this *gas.Component, values ...interface{}) (interface{}, error) {
					_el := this.Element().(*dom.Element)
					if _el == nil {
						return []interface{}{}, nil
					}

					var scrollTop, offsetHeight int
					if config.Direction {
						scrollTop = _el.ChildNodes()[0].ScrollTop()
						offsetHeight = _el.OffsetHeight()
					} else {
						scrollTop = _el.ChildNodes()[0].ScrollLeft()
						offsetHeight = _el.OffsetWidth()
					}

					endRaw, err := strconv.Atoi(fmt.Sprintf("%.0f", float64(offsetHeight)/float64(config.childSize)))
					if err != nil {
						return []interface{}{}, err
					}

					start := scrollTop / config.childSize
					end := start + endRaw //+ config.DefaultItemsSmooth

					err = this.SetValueImm("start", start)
					if err != nil {
						return []interface{}{}, err
					}

					err = this.SetValueImm("topPadding", start*config.childSize)
					if err != nil {
						return []interface{}{}, err
					}

					if config.Change != nil {
						err = config.Change(start, end)
						if err != nil {
							return []interface{}{}, err
						}
					}

					return getSlice(config.Items, start, end), nil
				},
				"genItems": func(this *gas.Component, values ...interface{}) (interface{}, error) {
					items := values[0].([]interface{})
					if len(items) == 0 {
						items = this.Method("calculateItems").([]interface{})
					}

					var _items []interface{}
					t := e.Templates["item"]
					for i, item := range items {
						_items = append(_items, t(i, item, this.Get("start").(int)))
					}

					return gas.NE(
						&gas.C{
							Tag: config.ItemsWrapperTag,
							Binds: map[string]gas.Bind{
								"style": func() string {
									var dFlex string
									if !config.Direction {
										dFlex = "display: flex;"
									}
									return fmt.Sprintf(`transform: %s(%dpx);%s`, config.directionTransform, this.Get(`topPadding`), dFlex)
								},
							},
							Attrs: map[string]string{
								"class": "vlist-content",
							},
						},
						_items...), nil
				},
			},
		},
		func(this *gas.Component) []interface{} {return gas.ToGetComponentList(gas.NE(
&gas.Component{Tag:"div",
Attrs: map[string]string{"class": "vlist",
},
Handlers: map[string]gas.Handler{
"scroll": func(p *gas.Component, e gas.Object) { this.Method(`onScroll`, e) },
},},
gas.NE(
&gas.Component{Tag:"div",
Attrs: map[string]string{"class": "vlist-padding",
},

Binds: map[string]gas.Bind{
"style": func() string {
	return fmt.Sprintf(`%s: %dpx;`, config.directionV, this.Get(`scrollHeight`))
},

},},),this.Method(`genItems`, this.Get(`items`)).(interface{}),),)})
}

var vlistT gas.GetComponentChildes

func getSlice(arr []interface{}, start, end int) []interface{} {
	if start > len(arr) {
		return []interface{}{}
	}

	if end > len(arr) {
		end = len(arr)
	}

	return arr[start:end]
}
