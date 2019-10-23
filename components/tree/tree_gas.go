package tree

import (
	"github.com/gascore/dom"
	"github.com/gascore/gas"
)

type Config struct {
	Name  string
	Items []*Item

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

func (config Config) Init() *gas.C {
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

  return c
}

type treeEl struct {
  c *gas.C

  isHidden bool
  config *Config
}

func (root *treeEl) Render() *gas.E {
  return gas.NE(&gas.E{Tag:"div", Attrs: func() gas.Map { return gas.Map{"class": "tree",} },},gas.NE(&gas.E{Tag:"div", Attrs: func() gas.Map { return gas.Map{"class": "tree-header",} },},``, root.config.Name , ),gas.NE(&gas.E{Tag:"ul", Attrs: func() gas.Map { return gas.Map{"class": "tree-items",} },},func()[]interface{}{var c8641486337242112828 []interface{}; for _, nItem := range root.config.Items { c8641486337242112828 = append(c8641486337242112828, gas.NE(&gas.E{Tag:"li", },renderItem(nItem, root.config),)) }; return c8641486337242112828}(),),)
}
