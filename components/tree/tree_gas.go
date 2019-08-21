package tree

import (
	"github.com/gascore/dom"
	"github.com/gascore/gas"
)

type Config struct {
	Name 		    string
	CanBeHidden bool
	Items       []*Item

  OnItemClick func(item *Item) error

  ArrowRight string
  ArrowDown  string
}

type Item struct {
	IsDirectory bool
  IsOpen      bool
	Childes     []*Item // if is directory

	Name     string
	Renderer func(*Item) *gas.E // if null render item name

	Data interface{} // Your custom data
}

func (config Config) Init() *gas.E {
  if config.Items == nil {
		dom.ConsoleError("invalid items")
		return nil
	}

  if len(config.ArrowRight) == 0 {
    config.ArrowRight = `<i class="icon-arrow-right icon"></i>`
  }

  if len(config.ArrowDown) == 0 {
    config.ArrowDown = `<i class="icon-arrow-down icon"></i>`
  }

  root := &treeEl{
    config: &config,
  }

  c := &gas.C{Root: root}
  root.c = c

  return c.Init()
}

type treeEl struct {
  c *gas.C

  isHidden bool
  config *Config
}

func (root *treeEl) Render() []interface{} {
  return gas.CL(gas.NE(&gas.E{Tag:"div", Attrs: func() gas.Map { return gas.Map{"class": "tree",} },},gas.NE(&gas.E{Tag:"div", Attrs: func() gas.Map { return gas.Map{"class": "tree-header",} },},gas.NE(&gas.E{Tag:"b", },``, root.config.Name , ),func()interface{} { if root.config.CanBeHidden { return gas.NE(&gas.E{Tag:"button", Handlers: map[string]gas.Handler{"click": func(e gas.Event) {root.toggleIsHidden() }, },Attrs: func() gas.Map { return gas.Map{"class": "tree-hide-btn",} },}, func()interface{} { if root.isHidden { return gas.NE(&gas.E{Tag:"span", },`                    Show                `,) } else { return gas.NE(&gas.E{Tag:"span", },`                    Hide                `,) }; return nil }(),) }; return nil }(),),func()interface{} { if !root.isHidden { return gas.NE(&gas.E{Tag:"ul", Attrs: func() gas.Map { return gas.Map{"class": "tree-items",} },},func()[]interface{}{var c7201249052817271062 []interface{}; for _, nItem := range root.config.Items { c7201249052817271062 = append(c7201249052817271062, gas.NE(&gas.E{Tag:"li", },renderItem(nItem, root.config),)) }; return c7201249052817271062}(),) }; return nil }(),),)
}

func (root *treeEl) toggleIsHidden() {
  if !root.config.CanBeHidden {
    return
  }

  root.isHidden = !root.isHidden
  go root.c.Update()
}
