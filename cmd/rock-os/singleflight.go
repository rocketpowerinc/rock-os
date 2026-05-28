package main

import "sync"

type requestFlightGroup struct {
	mu    sync.Mutex
	calls map[string]*requestFlightCall
}

type requestFlightCall struct {
	wg    sync.WaitGroup
	value any
	err   error
}

func newRequestFlightGroup() *requestFlightGroup {
	return &requestFlightGroup{
		calls: map[string]*requestFlightCall{},
	}
}

func (group *requestFlightGroup) Do(key string, fn func() (any, error)) (any, error) {
	group.mu.Lock()
	if call, ok := group.calls[key]; ok {
		group.mu.Unlock()
		call.wg.Wait()
		return call.value, call.err
	}

	call := &requestFlightCall{}
	call.wg.Add(1)
	group.calls[key] = call
	group.mu.Unlock()

	call.value, call.err = fn()
	call.wg.Done()

	group.mu.Lock()
	delete(group.calls, key)
	group.mu.Unlock()

	return call.value, call.err
}
