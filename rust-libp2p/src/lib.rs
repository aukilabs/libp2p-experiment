// Copyright 2018 Parity Technologies (UK) Ltd.
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS
// OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

#![doc = include_str!("../README.md")]

use futures::stream::StreamExt;
use libp2p::{gossipsub,swarm::{NetworkBehaviour,SwarmEvent},PeerId};
#[cfg(not(target_arch = "wasm32"))]
use libp2p::{mdns,noise,tcp,yamux,Transport,core::muxing::StreamMuxerBox,multiaddr::{Multiaddr, Protocol}};
use std::collections::{hash_map::DefaultHasher,HashMap};
use std::error::Error;
use std::hash::{Hash, Hasher};
use std::time::Duration;
#[cfg(not(target_arch = "wasm32"))]
use tokio::{io, select, time::interval, runtime::Runtime};
#[cfg(not(target_arch = "wasm32"))]
use tracing_subscriber::EnvFilter;
use serde::{Deserialize, Serialize};
use serde_json;
#[cfg(not(target_arch = "wasm32"))]
use rand::thread_rng;
#[cfg(not(target_arch = "wasm32"))]
use libp2p_webrtc as webrtc;
#[cfg(not(target_arch = "wasm32"))]
use std::net::Ipv4Addr;
#[cfg(target_arch = "wasm32")]
use libp2p_webrtc_websys as webrtc_websys;
#[cfg(target_arch = "wasm32")]
use wasm_bindgen::prelude::*;
#[cfg(target_arch = "wasm32")]
use std::io;

#[cfg(not(target_arch = "wasm32"))]
#[cxx::bridge]
mod ffi {
    extern "Rust" {
        // Synchronous wrapper for the async function
        fn start_libp2p_sync() -> String;
    }
}

// We create a custom network behaviour that combines Gossipsub and Mdns.
#[derive(NetworkBehaviour)]
struct MyBehaviour {
    gossipsub: gossipsub::Behaviour,
    #[cfg(not(target_arch = "wasm32"))]
    mdns: mdns::tokio::Behaviour,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Node {
    #[serde(with = "peer_id_serde")]
    id: PeerId,
    name: String,
    node_types: Vec<String>,   // Assuming node_types is a list of strings
    capabilities: Vec<String>, // Assuming capabilities is a list of strings
}

mod peer_id_serde {
    use libp2p::PeerId;
    use serde::{self, Deserialize, Deserializer, Serializer};
    use std::str::FromStr;

    pub fn serialize<S>(peer_id: &PeerId, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&peer_id.to_string())
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<PeerId, D::Error>
    where
        D: Deserializer<'de>,
    {
        let peer_id_string = String::deserialize(deserializer)?;
        PeerId::from_str(&peer_id_string).map_err(serde::de::Error::custom)
    }
}

// Function to create and serialize a node
fn create_node(peer_id: PeerId) -> Node {
    Node {
        id: peer_id,
        name: "VisionServer1".to_string(),
        node_types: vec!["VISION_SERVER".to_string()],
        capabilities: vec!["SHELF_SCAN".to_string()],
    }
}

#[cfg(not(target_arch = "wasm32"))]
pub fn start_libp2p_sync() -> String {
    let rt = Runtime::new().unwrap();

    match rt.block_on(start_libp2p()) {
        Ok(_) => "Success".to_string(),
        Err(e) => e.to_string(),
    }
}

#[cfg(target_arch = "wasm32")]
#[wasm_bindgen]
pub async fn start_libp2p() -> Result<(), JsError> {
    tracing_wasm::set_as_global_default();

    let mut swarm = libp2p::SwarmBuilder::with_new_identity()
        .with_wasm_bindgen()
        .with_other_transport(|key| {
            webrtc_websys::Transport::new(webrtc_websys::Config::new(&key))
        })?
        .with_behaviour(|key| {
            // To content-address message, we can take the hash of message and use it as an ID.
            let message_id_fn = |message: &gossipsub::Message| {
                let mut s = DefaultHasher::new();
                message.data.hash(&mut s);
                gossipsub::MessageId::from(s.finish().to_string())
            };

            // Set a custom gossipsub configuration
            let gossipsub_config = gossipsub::ConfigBuilder::default()
                .heartbeat_interval(Duration::from_secs(10)) // This is set to aid debugging by not cluttering the log space
                .validation_mode(gossipsub::ValidationMode::Strict) // This sets the kind of message validation. The default is Strict (enforce message signing)
                .message_id_fn(message_id_fn) // content-address messages. No two messages of the same content will be propagated.
                .build()
                .map_err(|msg| io::Error::new(io::ErrorKind::Other, msg))?; // Temporary hack because `build` does not return a proper `std::error::Error`.

            // build a gossipsub network behaviour
            let gossipsub = gossipsub::Behaviour::new(
                gossipsub::MessageAuthenticity::Signed(key.clone()),
                gossipsub_config,
            )?;

            Ok(MyBehaviour { gossipsub })
        })?
        .with_swarm_config(|c| c.with_idle_connection_timeout(Duration::from_secs(60)))
        .build();

    // Create a Gossipsub topic
    let topic = gossipsub::IdentTopic::new("Posemesh");
    // subscribes to our topic
    swarm.behaviour_mut().gossipsub.subscribe(&topic)?;

    // let address_webrtc = Multiaddr::from(Ipv4Addr::UNSPECIFIED)
    //     .with(Protocol::Udp(0))
    //     .with(Protocol::WebRTCDirect);

    // // Listen on all interfaces and whatever port the OS assigns
    // swarm.listen_on("/ip4/0.0.0.0/udp/0/quic-v1".parse()?)?;
    // swarm.listen_on("/ip4/0.0.0.0/tcp/0".parse()?)?;
    // swarm.listen_on(address_webrtc.clone())?;

    let mut connected_peers = 0;
    // Periodically check the peer count and publish if peers are available
    // let mut publish_check_interval = interval(Duration::from_secs(10));
    // let mut published = false;
    let mut nodes_map: HashMap<PeerId, Node> = HashMap::new();
    let peer_id = swarm.local_peer_id().clone();
    let node_info = create_node(peer_id);
    let serialized = serde_json::to_vec(&node_info)?;
    swarm.behaviour_mut().gossipsub.publish(topic.clone(), serialized)?;

    // Kick it off
    loop {
        // Publish node info when there are discovered peers
        // match publish_check_interval.tick() => {
        //     if connected_peers >= 1 && !published {  // Set your own threshold
                // match  {
                //     Ok(_) => {
                //         println!("Successfully published node info");
                //         // published = true;
                //     }
                //     Err(e) => {
                //         // Handle publish error
                //         eprintln!("Failed to publish node info: {}", e);
                //     }
                // }
        //     }
        // },
        match swarm.next().await.unwrap() {
            SwarmEvent::Behaviour(MyBehaviourEvent::Gossipsub(gossipsub::Event::Message {
                propagation_source: _peer_id,
                message_id: _id,
                message,
            })) => {
                match serde_json::from_slice::<Node>(&message.data) {
                    Ok(node) => {
                        println!("Got node info: {:?}", node);
                        nodes_map.insert(node.id.clone(), node);
                    },
                    Err(e) => {
                        println!("Failed to deserialize node info: {}", e);
                    }
                }
            },
            SwarmEvent::NewListenAddr { address, .. } => {
                println!("Local node is listening on {address}");
            },
            _ => {}
        }
        
    }
}

#[cfg(not(target_arch = "wasm32"))]
pub async fn start_libp2p() -> Result<(), Box<dyn Error>> {
    let _ = tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .try_init();

    let mut swarm = libp2p::SwarmBuilder::with_new_identity()
        .with_tokio()
        .with_tcp(
            tcp::Config::default(),
            noise::Config::new,
            yamux::Config::default,
        )?
        .with_quic()
        .with_other_transport(|id_keys| {
            Ok(webrtc::tokio::Transport::new(
                id_keys.clone(),
                webrtc::tokio::Certificate::generate(&mut thread_rng())?,
            )
            .map(|(peer_id, conn), _| (peer_id, StreamMuxerBox::new(conn))))
        })?
        .with_behaviour(|key| {
            // To content-address message, we can take the hash of message and use it as an ID.
            let message_id_fn = |message: &gossipsub::Message| {
                let mut s = DefaultHasher::new();
                message.data.hash(&mut s);
                gossipsub::MessageId::from(s.finish().to_string())
            };

            // Set a custom gossipsub configuration
            let gossipsub_config = gossipsub::ConfigBuilder::default()
                .heartbeat_interval(Duration::from_secs(10)) // This is set to aid debugging by not cluttering the log space
                .validation_mode(gossipsub::ValidationMode::Strict) // This sets the kind of message validation. The default is Strict (enforce message signing)
                .message_id_fn(message_id_fn) // content-address messages. No two messages of the same content will be propagated.
                .build()
                .map_err(|msg| io::Error::new(io::ErrorKind::Other, msg))?; // Temporary hack because `build` does not return a proper `std::error::Error`.

            // build a gossipsub network behaviour
            let gossipsub = gossipsub::Behaviour::new(
                gossipsub::MessageAuthenticity::Signed(key.clone()),
                gossipsub_config,
            )?;

            let mdns =
                mdns::tokio::Behaviour::new(mdns::Config::default(), key.public().to_peer_id())?;
            Ok(MyBehaviour { gossipsub, mdns })
        })?
        .with_swarm_config(|c| c.with_idle_connection_timeout(Duration::from_secs(60)))
        .build();

    // Create a Gossipsub topic
    let topic = gossipsub::IdentTopic::new("Posemesh");
    // subscribes to our topic
    swarm.behaviour_mut().gossipsub.subscribe(&topic)?;

    let address_webrtc = Multiaddr::from(Ipv4Addr::UNSPECIFIED)
        .with(Protocol::Udp(0))
        .with(Protocol::WebRTCDirect);

    // Listen on all interfaces and whatever port the OS assigns
    swarm.listen_on("/ip4/0.0.0.0/udp/0/quic-v1".parse()?)?;
    swarm.listen_on("/ip4/0.0.0.0/tcp/0".parse()?)?;
    swarm.listen_on(address_webrtc.clone())?;

    let mut connected_peers = 0;
    // Periodically check the peer count and publish if peers are available
    let mut publish_check_interval = interval(Duration::from_secs(10));
    let mut published = false;
    let mut nodes_map: HashMap<PeerId, Node> = HashMap::new();

    // Kick it off
    loop {
        select! {
            // Publish node info when there are discovered peers
            _ = publish_check_interval.tick() => {
                if connected_peers >= 1 && !published {  // Set your own threshold
                    let peer_id = swarm.local_peer_id().clone();
                    let node_info = create_node(peer_id);
                    let serialized = serde_json::to_vec(&node_info)?;
                    match swarm.behaviour_mut().gossipsub.publish(topic.clone(), serialized) {
                        Ok(_) => {
                            println!("Successfully published node info");
                            published = true;
                        }
                        Err(e) => {
                            // Handle publish error
                            eprintln!("Failed to publish node info: {}", e);
                        }
                    }
                }
            },
            event = swarm.select_next_some() => match event  {
                SwarmEvent::Behaviour(MyBehaviourEvent::Mdns(mdns::Event::Discovered(list))) => {
                    for (peer_id, _multiaddr) in list {
                        println!("mDNS discovered a new peer: {peer_id}");
                        swarm.behaviour_mut().gossipsub.add_explicit_peer(&peer_id);
                        connected_peers+=1;
                    }
                },
                SwarmEvent::Behaviour(MyBehaviourEvent::Mdns(mdns::Event::Expired(list))) => {
                    for (peer_id, _multiaddr) in list {
                        println!("mDNS discover peer has expired: {peer_id}");
                        swarm.behaviour_mut().gossipsub.remove_explicit_peer(&peer_id);
                        connected_peers-=1;
                    }
                },
                SwarmEvent::Behaviour(MyBehaviourEvent::Gossipsub(gossipsub::Event::Message {
                    propagation_source: _peer_id,
                    message_id: _id,
                    message,
                })) => {
                    match serde_json::from_slice::<Node>(&message.data) {
                        Ok(node) => {
                            println!("Got node info: {:?}", node);
                            nodes_map.insert(node.id.clone(), node);
                        },
                        Err(e) => {
                            println!("Failed to deserialize node info: {}", e);
                        }
                    }
                },
                SwarmEvent::NewListenAddr { address, .. } => {
                    println!("Local node is listening on {address}");
                },
                _ => {}
            }
        }
    }
}
