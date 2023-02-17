# Sama Virtual Machine (SamaVM)

_Content-Addressable Key-Value Store w/EIP-712 Compatibility and Fee-Based Metering_

This code is similar to [SpacesVM](https://github.com/ava-labs/spacesvm) but
does away with the hierarchical, authenticated namespace, user-specified
keys, and key expiry.

## Avalanche Subnets and Custom VMs
Avalanche is a network composed of multiple sub-networks (called [subnets][Subnet]) that each contain
any number of blockchains. Each blockchain is an instance of a
[Virtual Machine (VM)](https://docs.avax.network/learn/platform-overview#virtual-machines),
much like an object in an object-oriented language is an instance of a class. That is,
the VM defines the behavior of the blockchain where it is instantiated. For example,
[Coreth (EVM)][Coreth] is a VM that is instantiated by the
[C-Chain]. Likewise, one could deploy another instance of the EVM as their own blockchain (to take
this to its logical conclusion).

## AvalancheGo Compatibility
```
[v0.0.1] AvalancheGo@v1.9.4-1.9.5
[v0.0.2] AvalancheGo@v1.9.5-1.9.6
[v0.0.3] AvalancheGo@v1.9.6-1.9.7
[v0.0.4] AvalancheGo@v1.9.7-1.9.8
```

## Introduction
Just as [Coreth] powers the [C-Chain], SamaVM can be used to power its own
blockchain in an Avalanche [Subnet]. Instead of providing a place to execute Solidity
smart contracts, however, SamaVM enables content-addressable storage of arbitrary
keys/values using any [EIP-712] compatible wallet.

### Content-Addressable Key/Value Storage
All keys in SamaVM are keccak256 hashes (each of a unique value stored in
state). The max length of values is defined in genesis but typically ranges
between 64-200KB. Any number of values can be linked together to store files in
the > 100s of MBs range (as long as you have the `BLB` to pay for it).

### [EIP-712] Compatible
The canonical digest of a SamaVM transaction is [EIP-712] compliant, so any
Web3 wallet that can sign typed data can interact with SamaVM.

**[EIP-712] compliance in this case, however, does not mean that SamaVM
is an EVM or even an EVM derivative.** SamaVM is a new Avalanche-native VM written
from scratch to optimize for storage-related operations.

### Random Value Inclusion
To deter node operators from deleting data stored in state, each block header
includes the hash of a randomly selected state value concatenated with the parent blockID.
If values are pruned, node operators can't produce/verify blocks.

## Usage
_If you are interested in running the VM, not using it. Jump to [Running the
VM](#running-the-vm)._

### sama-cli
#### Install
```bash
git clone https://github.com/SamaNetwork/SamaVM.git;
cd samavm;
go install -v ./cmd/sama-cli;
```

#### Usage
```
SamaVM CLI

Usage:
  sama-cli [command]

Available Commands:
  activity     View recent activity on the network
  completion   Generate the autocompletion script for the specified shell
  create       Creates a new key in the default location
  genesis      Creates a new genesis in the default location
  help         Help about any command
  network      View information about this instance of the SamaVM
  resolve      Reads a value at key
  resolve-file Reads a file at a root and saves it to disk
  set          Writes a value to SamaVM
  set-file     Writes a file to SamaVM (using multiple keys)
  transfer     Transfers units to another address

Flags:
      --endpoint string           RPC endpoint for VM
  -h, --help                      help for sama-cli
      --private-key-file string   private key file path (default ".sama-cli-pk")
      --verbose                   Print verbose information about operations

Use "sama-cli [command] --help" for more information about a command.
```

##### Uploading Files
```
sama-cli set-file ~/Downloads/computer.gif -> 6fe5a52f52b34fb1e07ba90bad47811c645176d0d49ef0c7a7b4b22013f676c8
sama-cli resolve-file 6fe5a52f52b34fb1e07ba90bad47811c645176d0d49ef0c7a7b4b22013f676c8 computer_copy.gif
```

### [Golang SDK](https://github.com/SamaNetwork/SamaVM/client/client.go)

## Running the VM
To build the VM (and `sama-cli`), run `./scripts/build.sh`.

### Running a local network
[`scripts/run.sh`](scripts/run.sh) automatically installs [avalanchego], sets up a local network,
and creates a `samavm` genesis file. To build and run E2E tests, you need to set the variable `E2E` before it: `E2E=true ./scripts/run.sh 1.7.11`

_See [`tests/e2e`](tests/e2e) to see how it's set up and how its client requests are made._

```bash
# to startup a local cluster (good for development)
cd ${HOME}/go/src/github.com/SamaNetwork/SamaVM
./scripts/run.sh 1.7.11

# to run full e2e tests and shut down cluster afterwards
cd ${HOME}/go/src/github.com/SamaNetwork/SamaVM
E2E=true ./scripts/run.sh 1.7.11
```

```bash
# inspect cluster endpoints when ready
cat /tmp/avalanchego-v1.7.11/output.yaml
<<COMMENT
endpoint: /ext/bc/2VCAhX6vE3UnXC6s1CBPE6jJ4c4cHWMfPgCptuWS59pQ9vbeLM
logsDir: ...
pid: 12811
uris:
- http://localhost:56239
- http://localhost:56251
- http://localhost:56253
- http://localhost:56255
- http://localhost:56257
COMMENT

# ping the local cluster
curl --location --request POST 'http://localhost:61858/ext/bc/BJfusM2TpHCEfmt5i7qeE1MwVCbw5jU1TcZNz8MYUwG1PGYRL/public' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "samavm.ping",
    "params":{},
    "id": 1
}'
<<COMMENT
{"jsonrpc":"2.0","result":{"success":true},"id":1}
COMMENT

# resolve a path
curl --location --request POST 'http://localhost:61858/ext/bc/BJfusM2TpHCEfmt5i7qeE1MwVCbw5jU1TcZNz8MYUwG1PGYRL/public' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "samavm.resolve",
    "params":{
      "key": "0xd35882ae256d63123710cf8ab4343282d4a2c246281d3ff5e2b244744c8f7be4"
    },
    "id": 1
}'
<<COMMENT
{"jsonrpc":"2.0","result":{"exists":true, "value":"....", "valueMeta":{....}},"id":1}
COMMENT

# to terminate the cluster
kill 12811
```

### Deploying Your Own Network
Anyone can deploy their own instance of the SamaVM as a subnet on Avalanche.
All you need to do is compile it, create a genesis, and send a few txs to the
P-Chain.

You can do this by following the [subnet tutorial]
or by using the [subnet-cli].

[EIP-712]: https://eips.ethereum.org/EIPS/eip-712
[avalanchego]: https://github.com/ava-labs/avalanchego
[subnet tutorial]: https://docs.avax.network/build/tutorials/platform/subnets/create-a-subnet
[subnet-cli]: https://github.com/ava-labs/subnet-cli
[Coreth]: https://github.com/ava-labs/coreth
[C-Chain]: https://docs.avax.network/learn/platform-overview/#contract-chain-c-chain
[Subnet]: https://docs.avax.network/learn/platform-overview/#subnets

## Future Work
### Moderation
`SamaVM` does not include any built-in moderation mechanism to block/remove illicit
content. In the future, someone could implement an M-of-N governance contract
that can remove any value if it violates some code of conduct.

### Improved Access Proof
The current `AccessProof` mechanism is naive and gameable (seeded by the parent
block hash and index). In the future, someone could implement an on-chain VRF
that could be used as a more robust seed.
