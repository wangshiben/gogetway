package test_server

import "testing"

func TestSimpleGetServer(t *testing.T) {
	SimpleGetServer()
}
func TestSimplePostServer(t *testing.T) {
	SimplePostServer()
}

func TestSimpleStressHTTPServer(t *testing.T) {
	SimpleStressHTTPServer()
}
func TestSimpleTCPReceiver(t *testing.T) {
	SimpleTCPReceiver()
}
