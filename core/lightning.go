package core

import (
	"errors"
	"fmt"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/common/result"
	"github.com/scripttoken/script/crypto"
	"github.com/scripttoken/script/rlp"
)

//
// ------- AggregatedVotes ------- //
//

type Signatures map[common.Address]*crypto.Signature

// Copy creates a deep copy of the Signatures map.
func (s Signatures) Copy() Signatures {
	clone := make(Signatures, len(s))
	for addr, sig := range s {
		// Create a deep copy of the signature.
		// Assuming common.Bytes is a slice of bytes, we copy the slice using append.
		var dataCopy common.Bytes
		if sig != nil {
			dataCopy = append(common.Bytes(nil), sig.ToBytes()...) // using a getter or field access
		}

		newSig, err := crypto.SignatureFromBytes(dataCopy) // create a new signature from the copied bytes
		if err != nil {
			// Handle the error appropriately, for example, by logging it or returning it.
			fmt.Printf("Error copying signature for address %s: %v\n", addr.Hex(), err)
			continue
		}

		clone[addr] = newSig
	}
	return clone
}

func (this Signatures) Verify(msg *common.Bytes) bool {
	for addr, sig := range this {
		if !sig.Verify(*msg, addr) {
			return false
		}
	}
	return true
}

func (this Signatures) Has(addr common.Address) bool {
	if _, exists := this[addr]; exists {
		return true
	}
	return false
}

func (this Signatures) Add(addr common.Address, sig *crypto.Signature) {
	this[addr] = sig
}

// AggregatedVotes represents votes on a block.
type AggregatedVotes struct {
	Block      common.Hash // Hash of the block.
	Lightnings common.Hash // Hash of lightning candidate pool.
	Signatures Signatures
}

func NewAggregateVotes(block common.Hash, lightnings *AddressSet) *AggregatedVotes {
	return &AggregatedVotes{
		Block:      block,
		Lightnings: lightnings.Hash(),
		Signatures: make(Signatures),
	}
}

func (a *AggregatedVotes) String() string {
	return fmt.Sprintf("AggregatedVotes{Block: %s, Lightnings: %s}", a.Block.Hex(), a.Lightnings.Hex())
}

func (a *AggregatedVotes) Copy() *AggregatedVotes {
	clone := &AggregatedVotes{
		Block:      a.Block,
		Lightnings: a.Lightnings,
		Signatures: a.Signatures.Copy(),
	}
	return clone
}

// signBytes returns the bytes to be signed.
func (a *AggregatedVotes) signBytes() common.Bytes {
	tmp := &AggregatedVotes{
		Block:      a.Block,
		Lightnings: a.Lightnings,
	}
	b, _ := rlp.EncodeToBytes(tmp)
	return b
}

// Sign adds signer's signature. Returns false if signer has already signed.
func (this *AggregatedVotes) Sign(key *crypto.PrivateKey) bool {
	addr := key.PublicKey().Address()
	if this.Signatures.Has(addr) {
		return false // Already signed, do nothing.
	}
	sig, _ := key.Sign(this.signBytes())
	this.Signatures.Add(addr, sig)
	return true
}

// Merge creates a new aggregation that combines two vote sets. Returns nil, nil if input vote
// is a subset of current vote.
func (this *AggregatedVotes) Merge(other *AggregatedVotes) (*AggregatedVotes, error) {
	if this.Block != other.Block || this.Lightnings != other.Lightnings {
		return nil, errors.New("Cannot merge incompatible votes")
	}
	flag := false
	for addr, sig := range other.Signatures {
		if !this.Signatures.Has(addr) {
			this.Signatures.Add(addr, sig)
			flag = true
		}
	}
	if flag {
		// The other vote is a subset of current vote
		return nil, nil
	}
	return this, nil
}

// Abs returns the number of voted lightnings in the vote
func (this *AggregatedVotes) Abs() int {
	return len(this.Signatures)
	/*
	   ret := 0

	   	for i := 0; i < len(a.Multiplies); i++ {
	   		if a.Multiplies[i] != 0 {
	   			ret += 1
	   		}
	   	}

	   return ret
	*/
}

// Pick selects better vote from two votes.
func (a *AggregatedVotes) Pick(b *AggregatedVotes) (*AggregatedVotes, error) {
	if a.Block != b.Block || a.Lightnings != b.Lightnings {
		return nil, errors.New("Cannot compare incompatible votes")
	}
	if b.Abs() > a.Abs() {
		return b, nil
	}
	return a, nil
}

// Validate verifies the voteset.
func (this *AggregatedVotes) Validate(lightnings *AddressSet) result.Result {
	if lightnings.Hash() != this.Lightnings {
		return result.Error("lightnings hash mismatch: lightnings.Hash(): %s, vote.Lightnings: %s", lightnings.Hash().Hex(), this.Lightnings.Hex())
	}
	//	if len(a.Multiplies) != gcp.WithStake().Len() {
	//		return result.Error("multiplies size %d is not equal to gcp size %d", len(a.Multiplies), gcp.WithStake().Len())
	//	}
	if this.Signatures == nil {
		return result.Error("signatures cannot be nil")
	}
	//pubKeys := lightnings.PubKeys()
	//aggPubkey := bls.AggregatePublicKeysVec(pubKeys, a.Multiplies)
	msg := this.signBytes()
	if !this.Signatures.Verify(&msg) {
		return result.Error("signature verification failed")
	}
	return result.OK
}

/*
//
// ------- LightningCandidatePool ------- //
//

var (
	MinLightningStakeDeposit *big.Int

//	MinLightningStakeDeposit1000 *big.Int
)

func init() {
	// Each stake deposit needs to be at least 10,000 Script
	MinLightningStakeDeposit = new(big.Int).Mul(new(big.Int).SetUint64(1), new(big.Int).SetUint64(1e18))
}

type LightningCandidatePool struct {
	SortedLightnings []*Lightning // Lightnings sorted by holder address.
}

// NewLightningCandidatePool creates a new instance of LightningCandidatePool.
func NewLightningCandidatePool() *LightningCandidatePool {
	return &LightningCandidatePool{
		SortedLightnings: []*Lightning{},
	}
}

// Add inserts lightning into the pool; returns false if lightning is already added.
func (gcp *LightningCandidatePool) Add(g *Lightning) bool {
	k := sort.Search(gcp.Len(), func(i int) bool {
		return bytes.Compare(gcp.SortedLightnings[i].Holder.Bytes(), g.Holder.Bytes()) >= 0
	})

	if k == gcp.Len() {
		gcp.SortedLightnings = append(gcp.SortedLightnings, g)
		return true
	}

	// Lightning is already added.
	if gcp.SortedLightnings[k].Holder == g.Holder {
		return false
	}
	gcp.SortedLightnings = append(gcp.SortedLightnings, nil)
	copy(gcp.SortedLightnings[k+1:], gcp.SortedLightnings[k:])
	gcp.SortedLightnings[k] = g
	return true
}

// Remove removes a lightning from the pool; returns false if lightning is not found.
func (gcp *LightningCandidatePool) Remove(g common.Address) bool {
	k := sort.Search(gcp.Len(), func(i int) bool {
		return bytes.Compare(gcp.SortedLightnings[i].Holder.Bytes(), g.Bytes()) >= 0
	})

	if k == gcp.Len() || bytes.Compare(gcp.SortedLightnings[k].Holder.Bytes(), g.Bytes()) != 0 {
		return false
	}
	gcp.SortedLightnings = append(gcp.SortedLightnings[:k], gcp.SortedLightnings[k+1:]...)
	return true
}

// Contains checks if given address is in the pool.
func (gcp *LightningCandidatePool) Contains(g common.Address) bool {
	k := sort.Search(gcp.Len(), func(i int) bool {
		return bytes.Compare(gcp.SortedLightnings[i].Holder.Bytes(), g.Bytes()) >= 0
	})

	if k == gcp.Len() || gcp.SortedLightnings[k].Holder != g {
		return false
	}
	return true
}

// WithStake returns a new pool with withdrawn lightnings filtered out.
func (gcp *LightningCandidatePool) WithStake() *LightningCandidatePool {
	ret := NewLightningCandidatePool()
	for _, g := range gcp.SortedLightnings {
		// Skip if lightning dons't have non-withdrawn stake
		hasStake := false
		for _, stake := range g.Stakes {
			if !stake.Withdrawn {
				hasStake = true
				break
			}
		}
		if !hasStake {
			continue
		}

		ret.Add(g)
	}
	return ret
}

// GetWithHolderAddress returns the lightning node correspond to the stake holder in the pool. Returns nil if not found.
func (gcp *LightningCandidatePool) GetWithHolderAddress(addr common.Address) *Lightning {
	for _, g := range gcp.SortedLightnings {
		if g.Holder == addr {
			return g
		}
	}
	return nil
}

// Index returns index of a public key in the pool. Returns -1 if not found.
func (gcp *LightningCandidatePool) Index(pubkey *bls.PublicKey) int {
	for i, g := range gcp.SortedLightnings {
		if pubkey.Equals(g.Pubkey) {
			return i
		}
	}
	return -1
}

// PubKeys exports lightnings' public keys.
func (gcp *LightningCandidatePool) PubKeys() []*bls.PublicKey {
	ret := make([]*bls.PublicKey, gcp.Len())
	for i, g := range gcp.SortedLightnings {
		ret[i] = g.Pubkey
	}
	return ret
}

// Implements sort.Interface for Lightnings based on
// the Address field.
func (gcp *LightningCandidatePool) Len() int {
	return len(gcp.SortedLightnings)
}
func (gcp *LightningCandidatePool) Swap(i, j int) {
	gcp.SortedLightnings[i], gcp.SortedLightnings[j] = gcp.SortedLightnings[j], gcp.SortedLightnings[i]
}
func (gcp *LightningCandidatePool) Less(i, j int) bool {
	return bytes.Compare(gcp.SortedLightnings[i].Holder.Bytes(), gcp.SortedLightnings[j].Holder.Bytes()) < 0
}

// Hash calculates the hash of gcp.
func (gcp *LightningCandidatePool) Hash() common.Hash {
	raw, err := rlp.EncodeToBytes(gcp)
	if err != nil {
		logger.Panic(err)
	}
	return crypto.Keccak256Hash(raw)
}

func (gcp *LightningCandidatePool) DepositStake(source common.Address, holder common.Address, amount *big.Int, pubkey *bls.PublicKey, blockHeight uint64) (err error) {
	minLightningStake := MinLightningStakeDeposit
	//if blockHeight >= common.HeightLowerGNStakeThresholdTo1000 {
	//	minLightningStake = MinLightningStakeDeposit1000
	//}
	if amount.Cmp(minLightningStake) < 0 {
		return fmt.Errorf("Insufficient stake: %v", amount)
	}

	matchedHolderFound := false
	for _, candidate := range gcp.SortedLightnings {
		if candidate.Holder == holder {
			matchedHolderFound = true
			err = candidate.depositStake(source, amount)
			if err != nil {
				return err
			}
			break
		}
	}

	if !matchedHolderFound {
		newLightning := &Lightning{
			StakeHolder: NewStakeHolder(holder, []*Stake{NewStake(source, amount)}),
			Pubkey:      pubkey,
		}
		gcp.Add(newLightning)
	}
	return nil
}

func (gcp *LightningCandidatePool) WithdrawStake(source common.Address, holder common.Address, currentHeight uint64) error {
	matchedHolderFound := false
	for _, g := range gcp.SortedLightnings {
		if g.Holder == holder {
			matchedHolderFound = true
			_, err := g.withdrawStake(source, currentHeight)
			if err != nil {
				return err
			}
			break
		}
	}

	if !matchedHolderFound {
		return fmt.Errorf("No matched stake holder address found: %v", holder)
	}
	return nil
}

func (gcp *LightningCandidatePool) ReturnStakes(currentHeight uint64) []*Stake {
	returnedStakes := []*Stake{}

	// need to iterate in the reverse order, since we may delete elemements
	// from the slice while iterating through it
	for cidx := gcp.Len() - 1; cidx >= 0; cidx-- {
		g := gcp.SortedLightnings[cidx]
		numStakeSources := len(g.Stakes)
		for sidx := numStakeSources - 1; sidx >= 0; sidx-- { // similar to the outer loop, need to iterate in the reversed order
			stake := g.Stakes[sidx]
			if (stake.Withdrawn) && (currentHeight >= stake.ReturnHeight) {
				logger.Printf("Stake to be returned: source = %v, amount = %v", stake.Source, stake.Amount)
				source := stake.Source
				returnedStake, err := g.returnStake(source, currentHeight)
				if err != nil {
					logger.Errorf("Failed to return stake: %v, error: %v", source, err)
					continue
				}
				returnedStakes = append(returnedStakes, returnedStake)
			}
		}

		if len(g.Stakes) == 0 { // the candidate's stake becomes zero, no need to keep track of the candidate anymore
			gcp.Remove(g.Holder)
		}
	}
	return returnedStakes
}

//
// ------- Lightning ------- //
//

type Lightning struct {
	*StakeHolder
	Pubkey *bls.PublicKey `json:"-"`
}

func (g *Lightning) String() string {
	return fmt.Sprintf("{holder: %v, pubkey: %v, stakes :%v}", g.Holder, g.Pubkey.String(), g.Stakes)
}
*/
