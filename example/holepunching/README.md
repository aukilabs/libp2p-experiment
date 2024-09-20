# libp2p Hole Punching Example

## Overview
This project showcases how to implement hole punching in a libp2p network using Go. It leverages [AutoNAT](https://docs.libp2p.io/concepts/nat/autonat/) to detect whether a node is behind a NAT. If a node is behind NAT, it finds potential relays within the posemesh and reserves a slot for communication. Once the relay connection is established, the [Identify](https://github.com/libp2p/specs/blob/master/identify/README.md) protocol informs other peers that they can connect to you through this relay.

Additionally, [Identify](https://github.com/libp2p/specs/blob/master/identify/README.md) is used to discover your IP address, and having a public IP is essential to initiate hole punching.

Whenever a direct connection becomes possible, we will seamlessly switch the traffic from the relay connections to the direct connection.

- UPnP (Universal Plug and Play) will be used if possible to establish direct connections.
- If UPnP is not available, a relay node will act as the final fallback for connections.

### NAT vs. PAT Clarification
- **PAT (Port Address Translation)** is more common in real-world setups.
- **NAT (Network Address Translation)** is used on AWS EC2, even when public IPs are assigned.
- For successful hole punching behind PAT, the source port must remain constant.

For a detailed explanation of the hole punching concept within libp2p, refer to the [libp2p Hole Punching Documentation](https://docs.libp2p.io/concepts/nat/hole-punching/#hole-punching-in-libp2p).

## Setup

### 1. Modify the Activation Threshold
For testing with three nodes, set the `ActivationThres` value to 1. More details can be found in [Libp2p Issue #722](https://github.com/libp2p/go-libp2p/issues/722).

### 2. Start the Relay Node
Run a relay node on a publicly accessible server. It is recommended to specify a fixed port to avoid random selection on restarts:
```bash
IPFS_LOGGING=debug go run example/holepunching/relay/main.go --port 18804 --relay true --name <RELAY_NODE_NAME>
```

### 3. Start the Dialer Target (Behind NAT)
Start a node behind a NAT. This node will use DHT to discover the relay and reserve a slot:
```bash
IPFS_LOGGING=debug go run example/holepunching/pnode/main.go --name <PRIVATE_NODE_NAME> --bootstrap /ip4/$RELAYIP/udp/$RELAYPORT/quic-v1/p2p/$RELAYPEERID
```
- Upon successful reservation, you will see `adding new relay` in the logs. Node should push its new address in `/ip4/<RELAYIP>/udp/<RELAYPORT>/quic-v1/p2p/<RELAYPEERID>/p2p-circuit/p2p/<YOURPEERID>` format to other peers, tell them to connect to you through this address. To verify the relay connection, look for log entries containing the keyword `p2p-circuit`, which confirms that the node is using a relay for communication.
- When the hole-punch protocol is ready, the log will display `Starting holepunch protocol.`

### 4. Start the Dialer Node (Behind NAT)
Run a dialer node behind a NAT. This node will find the private node using DHT and connect to it through the relay:
```bash
IPFS_LOGGING=debug go run example/holepunching/dialer/main.go --name <DIALER_NODE_NAME> --bootstrap /ip4/$RELAYIP/udp/$RELAYPORT/quic-v1/p2p/$RELAYPEERID
```
- The dialer and dialer target can be started in any order.
- Hole punching may fail if the dialer hasn't observed its own public address, resulting in the dialer target's log entry `failed to open hole-punching stream: failed to negotiate protocol: protocols not supported: [/libp2p/dcutr]`.

### Notes:
- Before the dialer target (relay + hole punch) is ready, the dialer will repeatedly encounter `routing: not found` errors.

## Troubleshooting

### 1. Error: `skipping node that we recently failed to obtain a reservation with ...` in dial target logs
- **Cause**: The relay connection may not have been established correctly.
- **Resolution**: 
  - If you don’t see `adding new relay` in the dial target logs, ensure the relay node is running and check its logs for details.
  - If you encounter `disconnected from relay`, verify that the relay is still operational. When a relay is restarted, it can take up to an hour to reconnect to the same relay. To shorten this time, reduce the backoff duration:
    ```go
    // WithBackoff sets the time to wait after failing to obtain a reservation with a candidate.
    func WithBackoff(d time.Duration) Option {
        return func(c *config) error {
            c.backoff = d
            return nil
        }
    }
    ```
