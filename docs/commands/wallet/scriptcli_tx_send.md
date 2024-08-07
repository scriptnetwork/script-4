## scriptcli tx send

Send tokens

### Synopsis

Send tokens

```
scriptcli tx send [flags]
```

### Examples

```
scriptcli tx send --chain="scriptnet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --script=10 --spay=9 --seq=1
```

### Options

```
      --async           block until tx has been included in the blockchain
      --chain string    Chain ID
      --fee string      Fee (default "1000000000000wei")
      --from string     Address to send from
  -h, --help            help for send
      --path string     Wallet derivation path
      --seq uint        Sequence number of the transaction
      --spay string    SPAY amount (default "0")
      --script string    Script amount (default "0")
      --to string       Address to send to
      --wallet string   Wallet type (soft|nano|trezor) (default "soft")
```

### Options inherited from parent commands

```
      --config string   config path
```

### SEE ALSO

* [scriptcli tx](scriptcli_tx.md)	 - Manage transactions

###### Auto generated by spf13/cobra on 24-Apr-2019
