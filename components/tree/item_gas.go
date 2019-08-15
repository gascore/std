package tree

import (
    "github.com/gascore/gas"
)

func renderItem(item *Item, config *Config) *gas.E {
    f := &gas.F{}

    onClick := func() {
        item.IsOpen = !item.IsOpen

        if config.OnItemClick != nil {
            f.C.WarnError(config.OnItemClick(item))
        }

        go f.C.Update()
    }

    return f.Init(true, func() []interface{} {return gas.CL( func()interface{} { if item.IsDirectory { return gas.NE(&gas.E{Tag:"div", Attrs: func() gas.Map { return gas.Map{"class": "tree-item tree-directory-item",} },},gas.NE(&gas.E{Tag:"div", Handlers: map[string]gas.Handler{"click": func(e gas.Event) {onClick()},},Attrs: func() gas.Map { return gas.Map{"class": "tree-item-header",} },},gas.NE(&gas.E{Tag:"button", Attrs: func() gas.Map { return gas.Map{"class": "tree-item-header__hide-btn",} },}, func()interface{} { if !item.IsOpen { return gas.NE(&gas.E{Tag:"span", HTML: gas.HTMLDirective{Render: func() string { return config.ArrowRight},},},) } else { return gas.NE(&gas.E{Tag:"span", HTML: gas.HTMLDirective{Render: func() string { return config.ArrowDown},},},) }; return nil }(),),gas.NE(&gas.E{Tag:"span", Attrs: func() gas.Map { return gas.Map{"class": "tree-item-header_name",} },}, func()interface{} { if item.Renderer == nil { return gas.NE(&gas.E{Tag:"span", Attrs: func() gas.Map { return gas.Map{"class": "tree-item-header__name",} },},``, item.Name , ) } else { return gas.NE(&gas.E{Tag:"span", Attrs: func() gas.Map { return gas.Map{"class": "tree-item-header__byRenderer",} },},item.Renderer(item),) }; return nil }(),),),func()interface{} { if item.IsOpen { return gas.NE(&gas.E{Tag:"ul", Attrs: func() gas.Map { return gas.Map{"class": "tree-item-subs",} },},func()[]interface{}{var c5490739846379427904 []interface{}; for _, nItem := range item.Childes { c5490739846379427904 = append(c5490739846379427904, gas.NE(&gas.E{Tag:"li", Attrs: func() gas.Map { return gas.Map{"class": "tree-item-subs_item",} },},renderItem(nItem, config),)) }; return c5490739846379427904}(),) }; return nil }(),) } else { return gas.NE(&gas.E{Tag:"div", Handlers: map[string]gas.Handler{"click": func(e gas.Event) {onClick()},},Attrs: func() gas.Map { return gas.Map{"class": "tree-item tree-item_name-only",} },}, func()interface{} { if item.Renderer == nil { return gas.NE(&gas.E{Tag:"span", Attrs: func() gas.Map { return gas.Map{"class": "tree-item_name",} },},``, item.Name , ) } else { return gas.NE(&gas.E{Tag:"span", Attrs: func() gas.Map { return gas.Map{"class": "tree-item_byRenderer",} },},item.Renderer(item),) }; return nil }(),) }; return nil }(),)})
}
