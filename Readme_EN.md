# gogetway

## Introduction

`gogetway` is an open-source, high-performance traffic recording gateway that supports TCP traffic. It is designed for capturing TCP-based network traffic and provides customizable levels of TCP traffic replay.  
It supports a wide range of TCP-based protocols, including **HTTP/1.0–HTTP/2.0**, **WebSocket**, **SSH**, and more.

## Features

1. **Embeddable as a library**: Can be directly integrated into your projects for secondary development. Comprehensive integration documentation is provided (currently in progress).
2. **Ready-to-use deployment**: Pre-built installation packages are available on the Releases page for quick setup and immediate use (under development).
3. **Extremely low resource consumption**: Uses minimal CPU and memory during operation.

## Use Cases

1. **Honeypot / "Honey Badger" servers**: Record raw network traffic for security forensics and auditing.
2. **CTF challenge reproduction**: Capture interaction or attack traffic to facilitate debugging and replay of challenges.
3. **Canary testing and production traffic monitoring**: Record real user traffic for regression testing, behavior analysis, or validation.
4. **Monitoring non-HTTP TCP protocols**: Capable of capturing traffic from protocols like SSH, enhancing operational visibility and security auditing.
5. **Anomalous traffic circuit breaking**: Extensible to implement circuit-breaking logic that intercepts and blocks malicious or abnormal traffic in real time.

## Architecture Diagram

![Runtime Design](img%2FruntimeDesgin.png)

## Next Development Milestones

1. **Plugin-based architecture**: Introduce pluggable hooks at key processing nodes to enable user-defined extensions.
2. **Enhance traffic replay module**: Currently supports only traffic reading; full traffic replay functionality will be implemented.
3. **Built-in HTTP management server**: Embed a default HTTP server to provide a web-based management interface, improving out-of-the-box usability (planned for v2.0).