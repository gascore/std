package store

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gascore/gas"
)

// Store main structure
type Store struct {
	Data     map[string]interface{}
	Handlers map[string]Handler

	MiddleWares []MiddleWare

	OnCreate   []OnCreateHook
	BeforeEmit []BeforeEmitHook
	AfterEmit  []AfterEmitHook

	subs []*gas.Component
}

// MiddleWare let you do something before all events who have this (MiddleWare.Prefix) prefix.
//
// Example: { Prefix: "hello", Hook: func(s *Store) error { log.Println("Someone said hello }.
// This middleware will trigger on events: "helloMark", "helloElen", "helloArtem", "hello*etc*"
type MiddleWare struct {
	Prefix string
	Hook   func(s *Store, values []interface{}) error
}

// OnCreateHook called when store initializing
type OnCreateHook func(s *Store) error

// BeforeEmitHook called before event was processed
type BeforeEmitHook func(store *Store, eventName string, values []interface{}) error

// AfterEmitHook called after event was proccessed
type AfterEmitHook func(store *Store, eventName string, updatesMap map[string]interface{}, values []interface{}) error

// Handler - event handler with your stuff. Returns updatesData which will be appended to main store Data.
type Handler func(s *Store, values ...interface{}) (updatesMap map[string]interface{}, err error)

// New initialize new store
func New(s *Store) (*Store, error) {
	if s.OnCreate != nil {
		for _, create := range s.OnCreate {
			err := create(s)
			if err != nil {
				return nil, err
			}
		}
	}

	if s.Data == nil {
		return nil, errors.New("store data is nil")
	}

	return s, nil
}

// GetWithError return Store.Data value by query
func (s *Store) GetWithError(query string) (interface{}, error) {
	val, ok := s.Data[query]
	if !ok {
		return nil, fmt.Errorf("undefined value: %s", query)
	}

	return val, nil
}

// Get proxy for GetSafely with error ignoring
func (s *Store) Get(query string) interface{} {
	val, _ := s.GetWithError(query)
	return val
}

// Emit runs event from Store handlers
func (s *Store) Emit(query string, values ...interface{}) error {
	handler, ok := s.Handlers[query]
	if !ok {
		return fmt.Errorf("undefined event name: %s", query)
	}

	if handler == nil {
		return fmt.Errorf("invalid handler for event: %s", query)
	}

	if s.BeforeEmit != nil {
		for _, beforeEmit := range s.BeforeEmit {
			if err := beforeEmit(s, query, values); err != nil {
				return nil
			}
		}
	}

	for _, mw := range s.MiddleWares {
		if !strings.HasPrefix(query, mw.Prefix) {
			continue
		}

		if mw.Hook == nil {
			return fmt.Errorf("hook is nil in middleware with prefix '%s'", mw.Prefix)
		}

		err := mw.Hook(s, values)
		if err != nil {
			return err
		}
	}

	updatesMap, err := handler(s, values...)
	if err != nil {
		return err
	}

	if updatesMap == nil {
		return nil
	}

	err = s.UpdateStore(updatesMap)
	if err != nil {
		return err
	}

	if s.AfterEmit != nil {
		for _, afterEmit := range s.AfterEmit {
			if err := afterEmit(s, query, updatesMap, values); err != nil {
				return nil
			}
		}
	}

	return nil
}

// UpdateStore update Store by replacing fields from updatesMap to Store.data
func (s *Store) UpdateStore(updatesMap map[string]interface{}) error {
	for uKey, uValue := range updatesMap {
		oValue := s.Data[uKey]
		if oValue == nil {
			return fmt.Errorf("undefined field in Data: %s", uKey)
		}

		if reflect.TypeOf(uValue) != reflect.TypeOf(oValue) {
			return fmt.Errorf("uncompared fields: %T and %T", uValue, oValue)
		}

		s.Data[uKey] = uValue
	}

	return s.update()
}

// RegisterComponent register new component in store
func (s *Store) RegisterComponent(c *gas.C) *gas.Component {
	created := c.Hooks.Created
	c.Hooks.Created = func() error {
		isRoot, err := s.isRoot(c)
		if err != nil {
			return err
		}

		if isRoot {
			s.subs = append(s.subs, c)
		}

		if created != nil {
			err := created()
			if err != nil {
				return err
			}
		}

		return nil
	}

	willDestroy := c.Hooks.BeforeDestroy
	c.Hooks.BeforeDestroy = func() error {
		for i, elC := range s.subs {
			if c == elC {
				if i >= len(s.subs) {
					i = len(s.subs) - 1
				}

				s.subs = append(s.subs[:i], s.subs[i+1:]...)
			}
		}

		if willDestroy != nil {
			err := willDestroy()
			if err != nil {
				return err
			}
		}

		return nil
	}

	return c
}

// RC alias for Store.RegisterComponent
func (s *Store) RC(c *gas.Component) *gas.Component {
	return s.RegisterComponent(c)
}

// isRoot check if component have no RegisteredComponents which will update him after store updates
func (s *Store) isRoot(c *gas.Component) (bool, error) {
	if c.Element.Parent == nil { // it's root element
		return true, nil
	}

	parent := c.Element.ParentComponent()
	if parent == nil {
		return true, nil
	}

	for _, sub := range s.subs {
		changed, _, err := gas.Changed(sub, parent)
		if err != nil {
			return false, err
		}

		if !changed {
			return false, nil
		}
	}

	return s.isRoot(parent.Component)
}

// update run ForceUpdate for all subs
func (s *Store) update() error {
	for _, sub := range s.subs {
		if sub.Element.BEElement() == nil {
			return errors.New("element undefined")
		}

		err := sub.UpdateWithError()
		if err != nil {
			return err
		}
	}

	return nil
}
