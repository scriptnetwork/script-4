package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"
"runtime/debug"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/common/result"
	"github.com/scripttoken/script/crypto"
	"github.com/scripttoken/script/crypto/bls"
	"github.com/scripttoken/script/rlp"
)

//
// ------- AggregatedVotes ------- //
//

// AggregatedVotes represents votes on a block.
type AggregatedVotes struct {
	Block      common.Hash    // Hash of the block.
	Gcp        common.Hash    // Hash of lightning candidate pool.
	Multiplies []uint32       // Multiplies of each signer.
	Signature  *bls.Signature // Aggregated signiature.
}

func NewAggregateVotes(block common.Hash, gcp *LightningCandidatePool) *AggregatedVotes {
	return &AggregatedVotes{
		Block:      block,
		Gcp:        gcp.Hash(),
		Multiplies: make([]uint32, gcp.WithStake().Len()),
		Signature:  bls.NewAggregateSignature(),
	}
}

func (a *AggregatedVotes) String() string {
	return fmt.Sprintf("AggregatedVotes{Block: %s, Gcp: %s,  Multiplies: %v}", a.Block.Hex(), a.Gcp.Hex(), a.Multiplies)
}

// signBytes returns the bytes to be signed.
func (a *AggregatedVotes) signBytes() common.Bytes {
	tmp := &AggregatedVotes{
		Block: a.Block,
		Gcp:   a.Gcp,
	}
	b, _ := rlp.EncodeToBytes(tmp)
	return b
}

// Sign adds signer's signature. Returns false if signer has already signed.
func (a *AggregatedVotes) Sign(key *bls.SecretKey, signerIdx int) bool {
	if a.Multiplies[signerIdx] > 0 {
		// Already signed, do nothing.
		return false
	}

	a.Multiplies[signerIdx] = 1
	a.Signature.Aggregate(key.Sign(a.signBytes()))
	return true
}

// Merge creates a new aggregation that combines two vote sets. Returns nil, nil if input vote
// is a subset of current vote.
func (a *AggregatedVotes) Merge(b *AggregatedVotes) (*AggregatedVotes, error) {
	if a.Block != b.Block || a.Gcp != b.Gcp {
		return nil, errors.New("Cannot merge incompatible votes")
	}
	newMultiplies := make([]uint32, len(a.Multiplies))
	isSubset := true
	for i := 0; i < len(a.Multiplies); i++ {
		newMultiplies[i] = a.Multiplies[i] + b.Multiplies[i]
		if newMultiplies[i] < a.Multiplies[i] || newMultiplies[i] < b.Multiplies[i] {
			return nil, errors.New("Signiature multipliers overflowed")
		}
		if a.Multiplies[i] == 0 && b.Multiplies[i] != 0 {
			isSubset = false
		}
	}
	if isSubset {
		// The other vote is a subset of current vote
		return nil, nil
	}
	newSig := a.Signature.Copy()
	newSig.Aggregate(b.Signature)
	return &AggregatedVotes{
		Block:      a.Block,
		Gcp:        a.Gcp,
		Multiplies: newMultiplies,
		Signature:  newSig,
	}, nil
}

// Abs returns the number of voted lightnings in the vote
func (a *AggregatedVotes) Abs() int {
	ret := 0
	for i := 0; i < len(a.Multiplies); i++ {
		if a.Multiplies[i] != 0 {
			ret += 1
		}
	}
	return ret
}

// Pick selects better vote from two votes.
func (a *AggregatedVotes) Pick(b *AggregatedVotes) (*AggregatedVotes, error) {
	if a.Block != b.Block || a.Gcp != b.Gcp {
		return nil, errors.New("Cannot compare incompatible votes")
	}
	if b.Abs() > a.Abs() {
		return b, nil
	}
	return a, nil
}

// Validate verifies the voteset.
func (a *AggregatedVotes) Validate(gcp *LightningCandidatePool) result.Result {
	if gcp.Hash() != a.Gcp {
		return result.Error("gcp hash mismatch: gcp.Hash(): %s, vote.Gcp: %s", gcp.Hash().Hex(), a.Gcp.Hex())
	}
	if len(a.Multiplies) != gcp.WithStake().Len() {
		return result.Error("multiplies size %d is not equal to gcp size %d", len(a.Multiplies), gcp.WithStake().Len())
	}
	if a.Signature == nil {
		return result.Error("signature cannot be nil")
	}
	pubKeys := gcp.WithStake().PubKeys()
	aggPubkey := bls.AggregatePublicKeysVec(pubKeys, a.Multiplies)
	if !a.Signature.Verify(a.signBytes(), aggPubkey) {
		return result.Error("signature verification failed")
	}
	return result.OK
}

// Copy clones the aggregated votes
func (a *AggregatedVotes) Copy() *AggregatedVotes {
	clone := &AggregatedVotes{
		Block: a.Block,
		Gcp:   a.Gcp,
	}
	if a.Multiplies != nil {
		clone.Multiplies = make([]uint32, len(a.Multiplies))
		copy(clone.Multiplies, a.Multiplies)
	}
	if a.Signature != nil {
		clone.Signature = a.Signature.Copy()
	}

	return clone
}

//
// ------- LightningCandidatePool ------- //
//

var (
	MinLightningStakeDeposit *big.Int

//	MinLightningStakeDeposit1000 *big.Int
)

func init() {
	// Each stake deposit needs to be at least 10,000 Script
	//MinLightningStakeDeposit = new(big.Int).Mul(new(big.Int).SetUint64(10000), new(big.Int).SetUint64(1e18))
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
    logger.Debugf("TR-job309_REWARDS 00000 Stack trace:\n%s", debug.Stack())
	logger.Debugf("TR-job309_REWARDS 00000 LightningCandidatePool::Add engine::vote. %v", g)

	k := sort.Search(gcp.Len(), func(i int) bool {
		return bytes.Compare(gcp.SortedLightnings[i].Holder.Bytes(), g.Holder.Bytes()) >= 0
	})

	if k == gcp.Len() {
		gcp.SortedLightnings = append(gcp.SortedLightnings, g)
    	logger.Debugf("TR-job309_REWARDS 00001 LightningCandidatePool::Add (First Lightning). gcp/len=%v", gcp.Len())
		return true
	}

	// Lightning is already added.
	if gcp.SortedLightnings[k].Holder == g.Holder {
    	logger.Debug("TR-job309_REWARDS 00002 LightningCandidatePool::Add.")
		return false
	}
	gcp.SortedLightnings = append(gcp.SortedLightnings, nil)
	copy(gcp.SortedLightnings[k+1:], gcp.SortedLightnings[k:])
	gcp.SortedLightnings[k] = g
	logger.Debugf("TR-job309_REWARDS 00009 LightningCandidatePool::Add. gcp/len=%v", gcp.Len())
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

    logger.Debugf("TR-job309_REWARDS 00001 DepositStake. DepositStake")

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
        logger.Debugf("TR-job309_REWARDS 00001 gcp.Add New Lightning: %v", newLightning)
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
