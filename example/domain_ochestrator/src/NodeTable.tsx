import React, { useEffect } from 'react';
import { createNode } from './libp2p/libp2p';

const NodeTable = () => {
    const [nodes, setNodes] = React.useState(new Map());
    useEffect(() => {
      const fn = async () => {
        const node = await createNode()
        node.services.pubsub.addEventListener('message', (message) => {
          const enc = new TextDecoder("utf-8");
          const newNode = JSON.parse(enc.decode(message.detail.data))
          console.log('Received new node:', newNode)
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
  return (
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
        {Array.from(nodes.values()).map((node, index) => (
          <tr key={index}>
            <td>{node.id}</td>
            <td>{node.name}</td>
            <td>{node.node_types.join(', ')}</td>
            <td>{node.address}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
};

export default NodeTable;
