# gogetway

## Introduction

`gogetway` is an open-source, high-performance traffic recording gateway that supports TCP traffic. It is designed for capturing TCP-based network traffic and provides customizable levels of TCP traffic replay.  
It supports a wide range of TCP-based protocols, including **HTTP/1.0–HTTP/2.0**, **WebSocket**, **SSH**, and more.

## Features

1. **Embeddable as a library**: Can be directly integrated into your projects for secondary development. Comprehensive integration documentation is provided (currently in progress).
2. **Ready-to-use binaries**: Pre-built `tcp-proxy` binaries are available on the Releases page for out-of-the-box proxying, recording, and replay.
3. **Extremely low resource consumption**: Uses minimal CPU and memory during operation.

## Use Cases

1. **Honeypot / "Honey Badger" servers**: Record raw network traffic for security forensics and auditing.
2. **CTF challenge reproduction**: Capture interaction or attack traffic to facilitate debugging and replay of challenges.
3. **Canary testing and production traffic monitoring**: Record real user traffic for regression testing, behavior analysis, or validation.
4. **Monitoring non-HTTP TCP protocols**: Capable of capturing traffic from protocols like SSH, enhancing operational visibility and security auditing.
5. **Anomalous traffic circuit breaking**: Extensible to implement circuit-breaking logic that intercepts and blocks malicious or abnormal traffic in real time.

## Architecture Diagram

![Runtime Design](img%2FruntimeDesgin.png)

## Completed Capabilities

1. **Plugin / hook support at key stages**: Users can customize forwarding and recording behavior.
2. **Replay support is available**: Recorded TCP traffic can now be replayed to a target service.
3. **Bundled `tcp-proxy` binaries**: Ordinary users can use the packaged binary directly.

## tcp-proxy Binary Usage

`tcp-proxy` supports proxying, recording, replay, and passive mirroring.

### 1. Proxy and record traffic

```bash
./tcp-proxy -l :9000 -f 127.0.0.1:8080 -o ./traffic.log
```

### 2. Replay recorded traffic to a target service

```bash
./tcp-proxy -r -f 127.0.0.1:8080 -o ./traffic.log
```

### 3. Passive mirror and record traffic only

```bash
./tcp-proxy -l 8090 -io -o ./mirror.log
```

This mode does not occupy the observed service port and is suitable for passive mirroring.

### 4. Mirror and forward to another service

```bash
./tcp-proxy -l 8090 -f 127.0.0.1:18090 -o ./mirror.log
```

## Notes

1. To use the `-io` passive mirror mode, install `Npcap` on Windows or `libpcap` on Linux.
2. When using `-io`, run the program with sufficient packet-capture permissions.
3. Without `-io`, the proxy-record and replay modes can be used directly.

## Next Development Milestones

1. **Built-in HTTP management server**: Embed a default HTTP server to provide a web-based management interface, improving out-of-the-box usability (planned for v2.0).
