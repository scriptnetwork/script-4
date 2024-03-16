# Script Blockchain Ledger Protocol

The Script Blockchain Ledger is a Proof-of-Stake decentralized ledger designed for the video streaming industry. It powers the Script token economy which incentives end users to share their redundant bandwidth and storage resources, and encourage them to engage more actively with video platforms and content creators. The ledger employs a novel [multi-level BFT consensus engine](docs/multi-level-bft-tech-report.pdf), which supports high transaction throughput, fast block confirmation, and allows mass participation in the consensus process. Off-chain payment support is built directly into the ledger through the resource-oriented micropayment pool, which is designed specifically to achieve the “pay-per-byte” granularity for streaming use cases. Moreover, the ledger storage system leverages the microservice architecture and reference counting based history pruning techniques, and is thus able to adapt to different computing environments, ranging from high-end data center server clusters to commodity PCs and laptops. The ledger also supports Turing-Complete smart contracts, which enables rich user experiences for DApps built on top of the Script Ledger. For more technical details, please refer to our [technical whitepaper](docs/script-technical-whitepaper.pdf) and [2019 IEEE ICBC paper](https://arxiv.org/pdf/1911.04698.pdf) "Scalable BFT Consensus Mechanism Through Aggregated
Signature Gossip".

To learn more about the Script Network in general, please visit the **Script Documentation site**: https://docs.scripttoken.org/docs/what-is-script-network.

## Table of Contents
- [Setup](#setup)
- [Smart Contract and DApp Development on Script](#smart-contract-and-dapp-development-on-script)

## Setup

### Intall Go

Install Go and set environment variables `GOPATH` , `GOBIN`, and `PATH`. The current code base should compile with **Go 1.14.2**. On macOS, install Go with the following command

```
brew install go@1.14.1
brew link go@1.14.1 --force
```

### Build and Install

Next, clone this repo into your `$GOPATH`. The path should look like this: `$GOPATH/src/github.com/scripttoken/script`

```
git clone -b main --single-branch https://github.com/scriptnetwork/script4.git $GOPATH/src/github.com/scripttoken/script
export SCRIPT_HOME=$GOPATH/src/github.com/scripttoken/script
cd $SCRIPT_HOME
```

Now, execute the following commands to build the Script binaries under `$GOPATH/bin`. Two binaries `script` and `scriptcli` are generated. `script` can be regarded as the launcher of the Script Ledger node, and `scriptcli` is a wallet with command line tools to interact with the ledger.

```
export GO111MODULE=on
make install
```

#### Notes for Linux binary compilation
The build and install process on **Linux** is similar, but note that Ubuntu 18.04.4 LTS / Centos 8 or higher version is required for the compilation. 

#### Notes for Windows binary compilation
The Windows binary can be cross-compiled from macOS. To cross-compile a **Windows** binary, first make sure `mingw64` is installed (`brew install mingw-w64`) on your macOS. Then you can cross-compile the Windows binary with the following command:

```
make exe
```

You'll also need to place three `.dll` files `libgcc_s_seh-1.dll`, `libstdc++-6.dll`, `libwinpthread-1.dll` under the same folder as `script.exe` and `scriptcli.exe`.


### Run Unit Tests
Run unit tests with the command below
```
make test_unit
```

## Smart Contract and DApp Development on Script

Script provides full support for Turing-Complete smart contract, and is EVM compatible. To start developing on the Script Blockchain, please check out the following links:

### Smart Contracts
* Smart contract and DApp development Overview: [link here](https://docs.scripttoken.org/docs/turing-complete-smart-contract-support). 
* Tutorials on how to interact with the Script blockchain through [Metamask](https://docs.scripttoken.org/docs/web3-stack-metamask), [Truffle](https://docs.scripttoken.org/docs/web3-stack-truffle), [Hardhat](https://docs.scripttoken.org/docs/web3-stack-hardhat), [web3.js](https://docs.scripttoken.org/docs/web3-stack-web3js), and [ethers.js](https://docs.scripttoken.org/docs/web3-stack-hardhat).
* TNT20 Token (i.e. ERC20 on Script) integration guide: [link here](https://docs.scripttoken.org/docs/script-blockchain-tnt20-token-integration-guide).

### Local Test Environment Setup
* Launching a local scriptnet: [link here](https://docs.scripttoken.org/docs/launch-a-local-scriptnet).
* Command line tools: [link here](https://docs.scripttoken.org/docs/command-line-tool).
* Connect to the [Testnet](https://docs.scripttoken.org/docs/connect-to-the-testnet), and the [Mainnet](https://docs.scripttoken.org/docs/connect-to-the-mainnet).
* Node configuration: [link here](https://docs.scripttoken.org/docs/script-blockchain-node-configuration).

### API References
* Native RPC API references: [link here](https://docs.scripttoken.org/docs/rpc-api-reference).
* Ethereum RPC API support: [link here](https://docs.scripttoken.org/docs/web3-stack-eth-rpc-support).

