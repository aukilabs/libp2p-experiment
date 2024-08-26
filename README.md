# Private IPFS nodes
Nodes share the same swarm key can talk to each other.
## Generate swarm key
https://github.com/ahester57/ipfs-private-swarm?tab=readme-ov-file#2-generate-swarmkey
## Example
```
$ docker compose private-ipfs up
$ docker compose exec private-ipfs ipfs add -r /data/test-data

added QmR9r7U1yujMsFRR3qQzzKi3gNmCn5b8ffdsBoELtpBwHC test-data/large.type
added QmRKrcmm2PKqSWg37gQ1NQ1MWopP7aGDFHHeEYf5jPM4rZ test-data/name.type
added QmRRtYCpsaAg2Ffy9iQzRcYdH4EbqfoupdqWE7216yJP1M test-data/name.type2
added Qmdz1JJ1A4LCDEG1bQ6LTigcrtLBzK1JpyEew8qx3gV6Db test-data
```
You can't find `Qmdz1JJ1A4LCDEG1bQ6LTigcrtLBzK1JpyEew8qx3gV6Db` on public network. Check 
https://ipfs.io/ipfs/Qmdz1JJ1A4LCDEG1bQ6LTigcrtLBzK1JpyEew8qx3gV6Db
You can't load resources from public network.
```
$ docker compose exec private-ipfs ipfs get bafkreia3tfqergbwe4oklbm2eukrvcf6x3nhhsfadlhbuydx2ej2pwuqcm
```

# Data replication
https://ipfscluster.io/
## Create a cluster secret
https://ipfscluster.io/documentation/guides/security/#the-cluster-secret
Peers share the same cluster secret will consider to be in the same cluster, become the replicas of each other.
## Example
```
$ docker compose up cluster1

INFOcluster ipfs-cluster/cluster.go:764     ** IPFS Cluster is READY **

$ docker compose exec cluster0 ipfs-cluster-ctl add -r /data/test-data

added QmR9r7U1yujMsFRR3qQzzKi3gNmCn5b8ffdsBoELtpBwHC test-data/large.type
added QmRKrcmm2PKqSWg37gQ1NQ1MWopP7aGDFHHeEYf5jPM4rZ test-data/name.type
added QmRRtYCpsaAg2Ffy9iQzRcYdH4EbqfoupdqWE7216yJP1M test-data/name.type2
added Qmdz1JJ1A4LCDEG1bQ6LTigcrtLBzK1JpyEew8qx3gV6Db test-data

$ docker compose exec cluster1 ipfs-cluster-ctl pin ls

Qmdz1JJ1A4LCDEG1bQ6LTigcrtLBzK1JpyEew8qx3gV6Db |  | PIN | Repl. Factor: -1 | Allocations: [everywhere] | Recursive | Metadata: no | Exp: ∞ | Added: 2024-01-15 02:36:02
```
Content you added to cluster peer1 will be automatically pinned to other peers in the cluster

# Submit computing job
## Start bacalhau Requester Node
```
$ docker compose bacalhau up
```

## Submit jobs using cli
Example: 
```
$ docker compose exec bacalhau bacalhau docker run --gpu 1 ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1 -- python main.py --o ./outputs --p "cod swimming through data"
```

## Submit jobs using API
https://docs.bacalhau.org/references/api/jobs#create-job
