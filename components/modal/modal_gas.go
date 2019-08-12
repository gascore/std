package modal

import (
	"fmt"
	"github.com/gascore/dom"
	"github.com/gascore/gas"
)

type Config struct {
	IsActive func() bool
	Closer func()

	MaxHeight string
	MaxWidth string

	ClassName string

	DisableEvents bool
}

func (config *Config) Normalize() {
	if config.ClassName == "" {
		config.ClassName = "modal"
	}

	if config.MaxHeight == "" {
		config.MaxHeight = "75vh"
	}

	if config.MaxWidth == "" {
		config.MaxWidth = "75vh"
	}
}

type Modal struct {
	c *gas.C
	body []interface{}

	config *Config

	containerStyles string
}

func (root *Modal) Render() []interface{} {
	return gas.CL(func()interface{} {
if root.config.IsActive() {
	return gas.NE(&gas.E{Tag:"div", Attrs: func() map[string]string { return map[string]string{"class": root.modalWindowClass(),} },},gas.NE(&gas.E{Tag:"div", Handlers: map[string]gas.Handler{"click": func(e gas.Event) {root.disable()},},Attrs: func() map[string]string { return map[string]string{"class": root.overlayClasses(),} },},),gas.NE(&gas.E{Tag:"div", Attrs: func() map[string]string { return map[string]string{"class": "modal-window_container","style": root.containerStyles,} },},root.body,),)
}
return nil
}(),)
}

func (root *Modal) overlayClasses() string {
	var classIsActive string
	if root.config.IsActive() {
		classIsActive = "modal-window_overlay-active"
	}

	return "modal-window_overlay "+classIsActive
}

func (root *Modal) modalWindowClass() string {
	var classIsActive string
	if root.config.IsActive() {
		classIsActive = "modal-window-active"
	}

	return "modal-window "+classIsActive
}

func (root *Modal) disable() {
	if !root.config.DisableEvents {
    	root.config.Closer()
    }
}

func GetModal(config *Config) gas.DynamicElement {
	config.Normalize()

	root := &Modal{
		config: config,
		containerStyles: fmt.Sprintf("max-height: %s; max-width: %s;", config.MaxHeight, config.MaxWidth),
	}

	c := &gas.C{Root: root}
	root.c = c

	el := c.Init()
	return func(e gas.External) *gas.E {
		if len(e.Body) == 0 {
			dom.ConsoleError("invalid modal body") // just warn
		}

		root.body = e.Body
		return el
	}
}
