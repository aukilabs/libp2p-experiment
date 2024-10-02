import { useEffect, useState } from 'react';
import './App.css';
import NodeTable from './NodeTable'; 
import init, { start_libp2p } from 'rust-libp2p';
// import 'rust-libp2p/rust_libp2p_bg.wasm.d.ts';

function App() {
  const [libp2p, setLibp2p] = useState(null);
  useEffect(() => {
    const fn = async () => {
      console.log('App started');
      // const libp2p = await import ('rust-libp2p');
      // // init();
      // // setLibp2p(libp2p as any);
      // await libp2p.start_libp2p();
      init();
      await start_libp2p();
    }
    fn();
  }, []);
  return (
    <div className="App">
      <NodeTable />
    </div>
  );
}

export default App;
