package main

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/wangshiben/gogetway/getwayServer"
)

type liveObservedPacketSource struct {
	handles   []*pcap.Handle
	packets   chan gopacket.Packet
	done      chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

func newLiveObservedPacketSource(device string, snapLen int32, promiscuous bool, timeout time.Duration, bpfFilter string) (getwayServer.ObservedPacketSource, error) {
	if device == "" {
		return nil, errors.New("device is empty")
	}
	return newLiveObservedPacketSources([]string{device}, snapLen, promiscuous, timeout, bpfFilter)
}

func newLiveObservedPacketSources(devices []string, snapLen int32, promiscuous bool, timeout time.Duration, bpfFilter string) (getwayServer.ObservedPacketSource, error) {
	uniqueDevices := make([]string, 0, len(devices))
	seen := make(map[string]struct{}, len(devices))
	for _, device := range devices {
		if device == "" {
			continue
		}
		if _, ok := seen[device]; ok {
			continue
		}
		seen[device] = struct{}{}
		uniqueDevices = append(uniqueDevices, device)
	}
	if len(uniqueDevices) == 0 {
		return nil, errors.New("no capture devices configured")
	}

	source := &liveObservedPacketSource{
		handles: make([]*pcap.Handle, 0, len(uniqueDevices)),
		packets: make(chan gopacket.Packet, 1024),
		done:    make(chan struct{}),
	}
	openErrors := make([]error, 0)
	for _, device := range uniqueDevices {
		handle, err := pcap.OpenLive(device, snapLen, promiscuous, timeout)
		if err != nil {
			openErrors = append(openErrors, fmt.Errorf("open %s: %w", device, err))
			continue
		}
		if bpfFilter != "" {
			if err := handle.SetBPFFilter(bpfFilter); err != nil {
				handle.Close()
				openErrors = append(openErrors, fmt.Errorf("set filter on %s: %w", device, err))
				continue
			}
		}
		source.handles = append(source.handles, handle)
		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		source.wg.Add(1)
		go source.forwardPackets(packetSource.Packets())
	}
	if len(source.handles) == 0 {
		if len(openErrors) == 0 {
			return nil, errors.New("no capture device could be opened")
		}
		return nil, errors.Join(openErrors...)
	}
	return source, nil
}

func (s *liveObservedPacketSource) forwardPackets(packetCh <-chan gopacket.Packet) {
	defer s.wg.Done()
	for {
		select {
		case <-s.done:
			return
		case packet, ok := <-packetCh:
			if !ok {
				return
			}
			if packet == nil {
				continue
			}
			select {
			case <-s.done:
				return
			case s.packets <- packet:
			}
		}
	}
}

func newLiveObservedTCPPortSource(device string, observedPort uint16) (getwayServer.ObservedPacketSource, error) {
	if device == "" {
		return nil, errors.New("device is empty")
	}
	return newLiveObservedTCPPortSources([]string{device}, observedPort)
}

func newLiveObservedTCPPortSources(devices []string, observedPort uint16) (getwayServer.ObservedPacketSource, error) {
	filter := fmt.Sprintf("tcp port %d", observedPort)
	return newLiveObservedPacketSources(devices, 65535, true, pcap.BlockForever, filter)
}

func newAutoObservedTCPPortSource(observedPort uint16) (getwayServer.ObservedPacketSource, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, err
	}
	deviceNames := make([]string, 0, len(devices))
	for _, device := range devices {
		if device.Name == "" {
			continue
		}
		deviceNames = append(deviceNames, device.Name)
	}
	if len(deviceNames) == 0 {
		return nil, errors.New("no capture devices found")
	}
	return newLiveObservedTCPPortSources(deviceNames, observedPort)
}

func (s *liveObservedPacketSource) Packets() <-chan gopacket.Packet {
	return s.packets
}

func (s *liveObservedPacketSource) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
		for _, handle := range s.handles {
			handle.Close()
		}
		s.wg.Wait()
		close(s.packets)
	})
	return nil
}
