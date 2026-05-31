//go:build ignore

package getwayServer

// pcap-specific live capture has been moved out of getwayServer.
//
// getwayServer now exposes only the generic ObservedPacketSource abstraction
// through gopacket_mirror.go so embedders can supply their own packet source
// implementation.
//
// The bundled tcp-proxy binary carries the default pcap-backed implementation
// in cmd/tcp-proxy/live_capture.go for ordinary users on Windows and Linux.
