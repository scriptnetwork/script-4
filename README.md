# Script-4 - The script.tv blockchain

The Script Blockchain Ledger is a Proof-of-Stake decentralized ledger designed for the video streaming industry.
It powers the Script token economy which incentives end users to share their redundant bandwidth and storage resources,
and encourage them to engage more actively with video platforms and content creators.

The ledger employs a novel multi-level BFT consensus engine, which supports high transaction throughput,
fast block confirmation, and allows mass participation in the consensus process.

Off-chain payment support is built directly into the ledger through the resource-oriented micropayment pool,
which is designed specifically to achieve the “pay-per-byte” granularity for streaming use cases.

Moreover, the ledger storage system leverages the microservice architecture and reference counting based history pruning techniques,
and is thus able to adapt to different computing environments, ranging from high-end data center server clusters to commodity PCs and laptops.

The ledger also supports Turing-Complete smart contracts, which enables rich user experiences for DApps built on top of 
the Script Ledger. For more details, please refer to our [docs](https://docs.script.tv).

To learn more about the Script Network in general, visit [ScriptTV home page](https://script.tv).

### Tested Operating Systems:

* Debian GNU/Linux 
* Ubuntu GNU/Linux 

### Presequisites. Build Dependencies:

	sudo apt install build-essential golang bc

### Build and Install. Debian based Linux

For debug mode execute:

	bin/build_install

For release/optimized mode execute:

	bin/build_install release

### Shell environment vars

After invocation of bin/build_install the env file /tmp/tmp65958 is available.
This file can be used to define environmental variables useful for invoking binaries from the shell.

Variables defined are:

	GOPATH
	SCRIPT_HOME
	GO111MODULE
	GOBIN


### View/Load env vars

	cat /tmp/tmp65958   #View
	. /tmp/tmp65958     #Load
	echo $GOBIN         #Check


### Invoke binaries. e.g. script (governance) or scriptcli (wallet)

	#Produce config template ${HOME}/.scriptcli/config.yaml
	$GOBIN/script --config=${HOME}/.script init

	#Initiate gov daemon
	$GOBIN/script start --config=${HOME}/.script

	#Config for testnet v3
	cp v3/testnet/script/config.yaml ${HOME}/.script/

	#Inititate wallet daemon listening on port 10002 (testnet), or 11002 (mainnet
	REMOTERPCENDPOINT=http://127.0.0.1:10001/rpc $GOBIN/scriptcli daemon start --config=${HOME}/.scriptcli --port 10002

	$GOBIN/script --help
	$GOBIN/scriptcli --help

### Alternative setups

#### The 1-liner node pre-compiled installer

[https://download.script.tv](https://download.script.tv)

## Documentation

### Console-Client

* [script functions](https://github.com/scriptnetwork/script-4/tree/master/docs/commands/ledger)
* [scriptcli functions](https://github.com/scriptnetwork/script-4/tree/master/docs/commands/wallet)

### RPC



