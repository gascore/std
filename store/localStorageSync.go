package store

import (
	"errors"
	"github.com/gascore/gas/web"
	webStore "github.com/gascore/std/localStorage"
)

// LSSyncOnCreate syncronize data from localStorage and store data. Use in OnCreate hook 
func LSSyncOnCreate(s *Store) error {
	var localStorage = webStore.NewDataStore(webStore.JSONEncoding, web.GetLocalStore)

	var dataRaw interface{}
	err := localStorage.Get("data", &dataRaw)
	if err != nil && err != webStore.ErrNilValue {
		return err
	}

	if dataRaw == nil {
		return nil
	}

	data, ok := dataRaw.(map[string]interface{})
	if !ok {
		return errors.New("invalid data type")
	}

	// merge data from localStorage and default Data
	for key, value := range s.Data {
		if _, ok := data[key]; !ok {
			data[key] = value
		}
	}

	s.Data = data
	
	return nil
}

// LSSyncAfterEmit syncronize data from localStorage and store data. Use in AfterEmit hook
func LSSyncAfterEmit(s *Store, eventName string, updatesMap map[string]interface{}, values []interface{}) error {
	var localStorage = webStore.NewDataStore(webStore.JSONEncoding, web.GetLocalStore)

	err := localStorage.Set("data", s.Data)
	if err != nil {
		return err
	}

	return nil
}