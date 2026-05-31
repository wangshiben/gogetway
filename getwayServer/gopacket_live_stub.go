//go:build ignore

package getwayServer

// pcap-specific live capture has been moved out of getwayServer.
//
// Secondary developers should implement getwayServer.ObservedPacketSource in
// their own integration layer when they need a custom capture backend.
