//  Copyright (c) 2018 Cisco and/or its affiliates.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at:
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package reader

import "github.com/fsnotify/fsnotify"

// FileDBReaderMock allows to mock filesystem reader
type FileDBReaderMock struct {
	responses []*WhenResp
	respCurr  int
	respMax   int
	// Readers used during testing
	eventChan chan fsnotify.Event
}

// NewFileDBReaderMock creates new instance of the mock and initializes response list
func NewFileDBReaderMock() *FileDBReaderMock {
	return &FileDBReaderMock{
		responses: make([]*WhenResp, 0),
		eventChan: make(chan fsnotify.Event),
	}
}

// WhenResp is helper struct with single method call and desired response items
type WhenResp struct {
	methodName string
	items      []interface{}
}

// When defines name of the related method. It creates a new instance of WhenResp with provided method name and
// stores it to the mock.
func (mock *FileDBReaderMock) When(methodName string) *WhenResp {
	resp := &WhenResp{
		methodName: methodName,
	}
	mock.responses = append(mock.responses, resp)
	return resp
}

// ThenReturn receives array of items, which are desired to be returned in mocked method defined in "When". The full
// logic is:
// - When('someMethod').ThenReturn('values')
//
// Provided values should match return types of method. If method returns multiple values and only one is provided,
// mock tries to parse the value and returns it, while others will be nil or empty.
//
// If method is called several times, all cases must be defined separately, even if the return value is the same:
// - When('method1').ThenReturn('val1')
// - When('method1').ThenReturn('val1')
//
// All mocked methods are evaluated in same order they were assigned.
func (when *WhenResp) ThenReturn(item ...interface{}) {
	when.items = item
}

// Auxiliary method returns next return value for provided method as generic type
func (mock *FileDBReaderMock) getReturnValues(name string) (response []interface{}) {
	for i, resp := range mock.responses {
		if resp.methodName == name {
			// Remove used response but retain order
			mock.responses = append(mock.responses[:i], mock.responses[i+1:]...)
			return resp.items
		}
	}
	// Return empty response
	return
}

// IsValid mocks original method
func (mock *FileDBReaderMock) IsValid(path string) bool {
	items := mock.getReturnValues("IsValid")
	return items[0].(bool)
}

// PathExists mocks original method
func (mock *FileDBReaderMock) PathExists(path string) bool {
	items := mock.getReturnValues("PathExists")
	return items[0].(bool)
}

// ProcessFiles mocks original method
func (mock *FileDBReaderMock) ProcessFiles(path string) ([]*File, error) {
	items := mock.getReturnValues("ProcessFiles")
	if len(items) == 1 {
		switch typed := items[0].(type) {
		case []*File:
			return typed, nil
		case error:
			return []*File{}, typed
		}
	} else if len(items) == 2 {
		return items[0].([]*File), items[1].(error)
	}
	return []*File{}, nil
}

// Watch starts simulated watcher calling 'onEvent' and 'onClose' on demand
func (mock *FileDBReaderMock) Watch(paths []string, onEvent func(event fsnotify.Event, reader API), onClose func()) error {
	go func() {
		for {
			event, ok := <-mock.eventChan
			if !ok {
				onClose()
				return
			}
			onEvent(event, mock)
		}
	}()
	return nil
}

// ToString always returns empty string
func (mock *FileDBReaderMock) ToString() string {
	return ""
}

// Close closes event channel
func (mock *FileDBReaderMock) Close() error {
	close(mock.eventChan)
	return nil
}

// SendEvent simulates event from filesystem. Reader has to be initialized.
func (mock *FileDBReaderMock) SendEvent(event fsnotify.Event) {
	mock.eventChan <- event
	return
}
