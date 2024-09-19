# libp2p mDNS Example

## Overview
This example demonstrates how to use mDNS (Multicast DNS) in libp2p to discover peers on the same local network. mDNS allows nodes to discover each other without needing to rely on a distributed hash table (DHT) or a relay. It’s an effective way for peers on the same local network to establish direct connections in a decentralized manner. While DHT can take up to a minute to initialize, mDNS provides almost instant discovery.

### Key Benefits of mDNS:
- Faster Initialization: While DHT can take up to a minute to initialize, mDNS - provides almost instant discovery.
- Ideal for Latency-Sensitive Applications: Connecting to local nodes is significantly faster than connecting to remote nodes, making mDNS a great choice for latency-sensitive use cases.

### Latency Comparison:
- Local node ping (RTT) is less than 1ms.
- Remote private node via relay (from the holepunching example) has an RTT of around 300ms.
- Direct connection to a remote private node (from the holepunching example) has an RTT of around 200ms.

## Setup
```
go run example/mdns/main.go
```

This example starts two libp2p nodes that do not rely on bootstrap nodes. Instead, they discover each other using the mDNS discovery service.
