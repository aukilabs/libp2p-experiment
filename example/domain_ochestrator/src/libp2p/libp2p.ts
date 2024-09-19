import { createLibp2p } from 'libp2p'
import { gossipsub } from '@chainsafe/libp2p-gossipsub'
import { webSockets } from '@libp2p/websockets'
import { noise } from '@chainsafe/libp2p-noise'
import { yamux } from '@chainsafe/libp2p-yamux'
import { bootstrap } from '@libp2p/bootstrap'
import { identify } from '@libp2p/identify'
import { webRTC } from '@libp2p/webrtc'
import { circuitRelayTransport } from '@libp2p/circuit-relay-v2'
import { webTransport } from '@libp2p/webtransport'


const bootstrapList = [
  "/ip4/13.52.221.114/udp/18804/quic-v1/webtransport/certhash/uEiDHQLkYdqGEs1dBGAceUQgaaY9LMULyneVyZPPk0mkeng/certhash/uEiCEyjRhJ5xJgZ5jVTxH2rmDtqD0Gq34TlVDdbIiEqabAw/p2p/12D3KooWKZXUGZCg982NtauvYKfCLLc7BuQXHd4sCUVqhTEFvGPA",
  // "/ip4/127.0.0.1/udp/18804/quic-v1/webtransport/certhash/uEiDN9ud-Q1UHWkDewH6fpUl48QFqb9RAgXgAzPfk69I8qg/certhash/uEiBHxTEhzv9ugZxjYwCLmfYHRsCeEaNCAuH_K_oVYTSY-Q/p2p/12D3KooWBeuStnFAFdTcjn8HH8bu6VKiUXHy6fLHJ4bv5fDU2mi9",
  // "/ip4/192.168.3.21/udp/18805/quic-v1/webtransport/certhash/uEiDi_mr7COjeN8kqe3KFLhg5Gx1g1oRYarl2SbVuQunlig/certhash/uEiAYgutxERsTglaucyxuFpmKAu3AXoypBhJXjk8d8w-qpg/p2p/12D3KooWLQUrSJJ8PvZ3iWT5mw14qiiuux8eV6irZMZZNDc368xk"
]

async function createNode() {
  // enable verbose logging in browser console to view debug logs
  localStorage.debug = 'ui*,libp2p*,-libp2p:connection-manager*,-*:trace'
  const node = await createLibp2p({
    addresses: {
      listen: [
        '/webrtc',
        // ...bootstrapList
      ]
    },
    transports: [
      webTransport(),
      webSockets(),
      webRTC({
        rtcConfiguration: {
          iceServers: [
            {
              // STUN servers help the browser discover its own public IPs
              urls: ['stun:stun.l.google.com:19302', 'stun:global.stun.twilio.com:3478'],
            },
          ],
        },
      }),
    circuitRelayTransport({discoverRelays: 1})],
    connectionEncryption: [noise()],
    connectionManager: {
      maxConnections: 30,
      minConnections: 5,
    },
    connectionGater: {
      denyDialMultiaddr: async () => false,
    },
    streamMuxers: [yamux()],
    peerDiscovery: [
      bootstrap({
        list: bootstrapList
      })
    ],
    services: {
      // dht: kadDHT({
      //   protocol: "/posemesh/kad/1.0.0",
      // }),
      identify: identify(),
      pubsub: gossipsub({
        emitSelf: true
      })
    }
  })

  // await node.start()
  // const peerId = peerIdFromString("12D3KooWLarz8bTotktq3UXPxWGxtbWUb9SRxeCTnWppAMt9eXkr")
  // const peerInfo = await node.peerRouting.findPeer(peerId, {
  //   signal: AbortSignal.timeout(5000),
  // })
  // console.log('Peer info:', peerInfo)
  // console.log('Dialing bootstrap node:', multiaddr(bootstrapList[0]))
  // await node.dial(multiaddr(bootstrapList[0]), {
  //   signal: AbortSignal.timeout(5000)
  // })
  
  // console.log('Node started:', node.peerId.toString())
  return node
}

export { createNode }
