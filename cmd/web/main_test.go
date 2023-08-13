package main

import "testing"

//test coverage command: go test -coverprofile=coverage; go tool cover -html=coverage
func TestRun(t *testing.T) {
	err := run()
	if err != nil {
		t.Error("failed run()")
	}
}

