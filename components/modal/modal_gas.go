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

	Background string
	ClassName string

	DisableEvents bool
}

func (config *Config) Normalize() {
	if config.ClassName == "" {
		config.ClassName = "modal"
	}

	if config.Background == "" {
		config.Background = "rgba(247,248,249,.75)"
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

	overlayStyles string
	containerStyles string
}

func (root *Modal) Render() []interface{} {
	return gas.CL(func()interface{} {
if root.config.IsActive() {
	return gas.NE(&gas.E{Tag:"div", Binds: map[string]gas.Bind{"class": func() string { return (root.modalWindowClass())},},},gas.NE(&gas.E{Tag:"div", Binds: map[string]gas.Bind{"class": func() string { return (root.overlayClasses())},"style": func() string { return (root.overlayStyles)},},Handlers: map[string]gas.Handler{"click": func(e gas.Object) {root.disable()},},},),gas.NE(&gas.E{Tag:"div", Binds: map[string]gas.Bind{"style": func() string { return (root.containerStyles)},},Attrs: map[string]string{"class": "modal-window_container",},},root.body,),)
}
return nil
}(),)
}

func (root *Modal) overlayClasses() string {
	var classIsActive string
	if root.config.IsActive() {
		classIsActive = "modal-window_overlay-active"
	}

	return `modal-window_overlay `+classIsActive
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
		overlayStyles: fmt.Sprintf("background: %s;", config.Background),
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
