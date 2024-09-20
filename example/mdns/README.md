# libp2p mDNS Example

## Overview
This example demonstrates how to use mDNS (Multicast DNS) in libp2p to discover peers on the same local network. mDNS allows nodes to discover each other without needing to rely on a distributed hash table (DHT) or a relay. It’s an effective way for peers on the same local network to establish direct connections in a decentralized manner. While DHT can take up to a minute to initialize, mDNS provides almost instant discovery.

### Key Benefits of mDNS:
- Faster Initialization: While DHT can take up to a minute to initialize, mDNS - provides almost instant discovery.
- Ideal for Latency-Sensitive Applications: Connecting to local nodes is significantly faster than connecting to remote nodes, making mDNS a great choice for latency-sensitive use cases.

### Latency Comparison:
- Local node UDP ping (RTT) is less than 1ms.
- UDP pinging remote private node via relay (from the holepunching example) has an RTT of around 300ms.
- UDP direct connection to a remote private node (from the holepunching example) has an RTT of around 200ms.

### Use Case Example:
Imagine we have deployed a few motion servers, vision servers, and a Cactus backend on the same network. When THC joins that network, it can quickly set up direct connections with the currently available motion and vision servers. This allows THC to begin streaming frames to those nodes with minimal delay and less effort in infrastructure—there’s no need to hard-code motion server and vision server URLs or worry about nodes going down or coming back up, as THC will dynamically connect to whichever nodes are available.

## Setup
```
go run example/mdns/main.go
```

This example starts two libp2p nodes that do not rely on bootstrap nodes. Instead, they discover each other using the mDNS discovery service.
