package modal

import (
	"fmt"
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

func GetModal(config *Config) gas.DynamicElement {
	config.Normalize()
	containerStyles := fmt.Sprintf("max-height: %s; max-width: %s;", config.MaxHeight, config.MaxWidth)

	f := &gas.F{}

	modalWindowClass := func() string {
		var classIsActive string
		if config.IsActive() {
			classIsActive = "modal-window-active"
		} else {
			classIsActive = "modal-window-hidden"
		}

		return "modal-window "+classIsActive
	}

	disable := func() {
		if !config.DisableEvents {
			config.Closer()
		}
	}

	return func(e gas.External) *gas.E {
		return f.Init(true, func() []interface{} {return gas.CL(gas.NE(&gas.E{Tag:"div", Attrs: func() gas.Map { return gas.Map{"class": modalWindowClass(),} },},gas.NE(&gas.E{Tag:"div", Handlers: map[string]gas.Handler{"click": func(e gas.Event) {disable() }, },Attrs: func() gas.Map { return gas.Map{"class": "modal-window_overlay",} },},),gas.NE(&gas.E{Tag:"div", Attrs: func() gas.Map { return gas.Map{"class": "modal-window_container","style": containerStyles,} },},e.Body,),),)})
	}
}
