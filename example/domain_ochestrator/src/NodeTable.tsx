import React, { useEffect } from 'react';
import { createNode } from './libp2p/libp2p';
import { Libp2p } from 'libp2p';
import { peerIdFromString } from '@libp2p/peer-id';
import { lpStream } from 'it-length-prefixed-stream';

const NodeTable = () => {
    const [nodes, setNodes] = React.useState(new Map());
    let libp2pRef = React.useRef<Libp2p|null>(null);
    useEffect(() => {
      const fn = async () => {
        const node = await createNode()
        libp2pRef.current = node
        node.services.pubsub.addEventListener('message', (message) => {
          const enc = new TextDecoder("utf-8");
          const newNode = JSON.parse(enc.decode(message.detail.data))
          console.log('Received new node:', newNode)
          // if (newNode.node_types[0] == "DOMAIN_SERVICE") {
          //   console.log('Received new domain service:', newNode)
          //   return
          // }
          setNodes((prevNodes) => {
              const newNodes = new Map(prevNodes)
              newNodes.set(newNode.id, newNode)
              return newNodes
          })
        })
        
        node.services.pubsub.subscribe('posemesh_nodes')
      }
        fn() 
    }, []);
  return (<>
    <button onClick={async () => {
      if (libp2pRef.current) {
        let dataNode: string = ""
        let discoveryNode: string = ""
        const otherNodes: any[] = []
        nodes.forEach((node) => {
          if (node.node_types.includes("DATA")) {
            dataNode = node
          } else if (node.node_types.includes("DOMAIN_SERVICE")) {
            discoveryNode = node.id
          } else {
            otherNodes.push(node)
          }
        })
        if (dataNode == "") {
          console.error('No data node found')
          return
        }
        if (discoveryNode == "") {
          console.error('No discovery node found')
          return
        }

        const dis = peerIdFromString(discoveryNode)
        const stream = await libp2pRef.current.dialProtocol(dis, "/posemesh/create_domain/1.0.0")

        const data = JSON.stringify({
          name: 'test',
          cluster: {
            writer: dataNode,
            others: otherNodes
          }
        })
        await stream.sink([new TextEncoder().encode(data)])

        // const lp = lpStream(stream)

        // await lp.write(new TextEncoder().encode(data))

        // // read the response
        // const res = await lp.read()
        // const output = JSON.parse(new TextDecoder().decode(res.subarray()))

        // console.info(`Domain: "${output.id}" created`)
      }
    }}>Create Domain</button>
    <table className="nodes">
      <thead>
        <tr>
          <th>ID</th>
          <th>Name</th>
          <th>Node Types</th>
          <th>Address</th>
        </tr>
      </thead>
      <tbody>

        {Array.from(nodes.values()).map((node, index) => node.node_types[0] == "DOMAIN_SERVICE" ? null : (
          <div key={index}>
            <input type='checkbox' onChange={(e) => {node.selected = !node.selected}} value={node.selected}/>
          <tr>
            <td>{node.id}</td>
            <td>{node.name}</td>
            <td>{node.node_types.join(', ')}</td>
            <td>{node.address}</td>
          </tr>
          </div>
        ))}
      </tbody>
    </table>
    </>
  );
};

export default NodeTable;
