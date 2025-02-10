package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/common/result"
	"github.com/scripttoken/script/crypto"
	"github.com/scripttoken/script/rlp"
	log "github.com/sirupsen/logrus"
)

// Proposal represents a proposal of a new block.
type Proposal struct {
	Block      *Block `rlp:"nil"`
	ProposerID common.Address
	Votes      *VoteSet `rlp:"nil"`
}

func (p Proposal) String() string {
	return fmt.Sprintf("Proposal{block: %v, proposer: %v, votes: %v}", p.Block, p.ProposerID, p.Votes)
}

// CommitCertificate represents a commit made a majority of validators.
type CommitCertificate struct {
	Votes     *VoteSet `rlp:"nil"`
	BlockHash common.Hash
}

// Copy creates a copy of this commit certificate.
func (cc CommitCertificate) Copy() CommitCertificate {
	ret := CommitCertificate{
		BlockHash: cc.BlockHash,
	}
	if cc.Votes != nil {
		ret.Votes = cc.Votes.Copy()
	}
	return ret
}

func (cc CommitCertificate) String() string {
	return fmt.Sprintf("CC{BlockHash: %v, Votes: %v}", cc.BlockHash.Hex(), cc.Votes)
}

func (s *AddressSet) HasMajority(v *VoteSet) bool {
	return len(v.votes)*3 > len(*s)*2
}

// IsValid checks if a CommitCertificate is valid.
func (cc CommitCertificate) IsValid(validators *AddressSet) bool {
	if cc.Votes == nil || cc.Votes.IsEmpty() {
		return false
	}
	filtered := cc.Votes.UniqueVoter()
	if filtered.Size() != cc.Votes.Size() {
		return false
	}
	if filtered.Size() > len(*validators) {
		return false
	}
	for _, vote := range filtered.Votes() {
		if vote.Block != cc.BlockHash {
			return false
		}
		if vote.Validate().IsError() {
			return false
		}
	}
	//return validators.HasMajority(filtered)
	return validators.HasMajority(filtered)
}

// Vote represents a vote on a block by a validaor.
type Vote struct {
	Block     common.Hash    // Hash of the tip as seen by the voter.
	Height    uint64         // Height of the tip
	Epoch     uint64         // Voter's current epoch. It doesn't need to equal the epoch in the block above.
	ID        common.Address // Voter's address.
	Signature *crypto.Signature
}

func (this Vote) String() string {
	return fmt.Sprintf("Vote{block: %s, Height: %s, Epoch: %v}", this.Block.Hex(), this.Height, this.Epoch)
}

// SignBytes returns raw bytes to be signed.
func (this Vote) SignBytes() common.Bytes {
	vv := Vote{
		Block: this.Block,
		Epoch: this.Epoch,
		ID:    this.ID,
	}
	raw, _ := rlp.EncodeToBytes(vv)
	return raw
}

// Sign signs the vote using given private key.
func (this *Vote) Sign(priv *crypto.PrivateKey) {
	sig, err := priv.Sign(this.SignBytes())
	if err != nil {
		// Should not happen.
		logger.WithFields(log.Fields{"error": err}).Panic("Failed to sign vote")
	}
	this.SetSignature(sig)
}

// SetSignature sets given signature in vote.
func (this *Vote) SetSignature(sig *crypto.Signature) {
	this.Signature = sig
}

// Validate checks the vote is legitimate.
func (this Vote) Validate() result.Result {
	if this.Block.IsEmpty() {
		return result.Error("Block is not specified")
	}
	if this.ID.IsEmpty() {
		return result.Error("Voter is not specified")
	}
	if this.Signature == nil || this.Signature.IsEmpty() {
		return result.Error("Vote is not signed")
	}
	if !this.Signature.Verify(this.SignBytes(), this.ID) {
		return result.Error("Signature verification failed")
	}
	return result.OK
}

// Hash calculates vote's hash.
func (v Vote) Hash() common.Hash {
	raw, _ := rlp.EncodeToBytes(v)
	return crypto.Keccak256Hash(raw)
}

// VoteSet represents a set of votes on a proposal.
type VoteSet struct {
	votes map[string]Vote // Voter ID to vote
}

// NewVoteSet creates an instance of VoteSet.
func NewVoteSet() *VoteSet {
	return &VoteSet{
		votes: make(map[string]Vote),
	}
}

// Copy creates a copy of this vote set.
func (s *VoteSet) Copy() *VoteSet {
	ret := NewVoteSet()
	for _, vote := range s.Votes() {
		ret.AddVote(vote)
	}
	return ret
}

// AddVote adds a vote to vote set. Duplicate votes are ignored.
func (s *VoteSet) AddVote(vote Vote) {
	key := fmt.Sprintf("%s:%s:%d", vote.ID, vote.Block, vote.Epoch)
	s.votes[key] = vote
}

// Size returns the number of votes in the vote set.
func (s *VoteSet) Size() int {
	return len(s.votes)
}

// IsEmpty returns wether the vote set is empty.
func (s *VoteSet) IsEmpty() bool {
	return s.Size() == 0
}

// Votes return a slice of votes in the vote set.
func (s *VoteSet) Votes() []Vote {
	ret := make([]Vote, 0, len(s.votes))
	for _, v := range s.votes {
		ret = append(ret, v)
	}
	sort.Sort(VoteByID(ret))
	return ret
}

// Validate checks the vote set is legitimate.
func (s *VoteSet) Validate() result.Result {
	for _, vote := range s.votes {
		if vote.Validate().IsError() {
			return result.Error("Contains invalid vote: %s", vote.String())
		}
	}
	return result.OK
}

func (s *VoteSet) String() string {
	if s == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", s.Votes())
}

// MarshalJSON implements json.Marshaler
func (s *VoteSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Votes())
}

// UnmarshalJSON implements json.Marshaler
func (s *VoteSet) UnmarshalJSON(b []byte) error {
	votes := []Vote{}
	if err := json.Unmarshal(b, &votes); err != nil {
		return err
	}
	s.votes = make(map[string]Vote)
	for _, v := range votes {
		s.AddVote(v)
	}
	return nil
}

var _ rlp.Encoder = (*VoteSet)(nil)

// EncodeRLP implements RLP Encoder interface.
func (s *VoteSet) EncodeRLP(w io.Writer) error {
	if s == nil {
		return rlp.Encode(w, []Vote{})
	}
	return rlp.Encode(w, s.Votes())
}

var _ rlp.Decoder = (*VoteSet)(nil)

// DecodeRLP implements RLP Decoder interface.
func (s *VoteSet) DecodeRLP(stream *rlp.Stream) error {
	votes := []Vote{}
	err := stream.Decode(&votes)
	if err != nil {
		return err
	}
	s.votes = make(map[string]Vote)
	for _, v := range votes {
		s.AddVote(v)
	}
	return nil
}

// Merge combines two vote sets.
func (s *VoteSet) Merge(another *VoteSet) *VoteSet {
	ret := NewVoteSet()
	for _, vote := range s.Votes() {
		ret.AddVote(vote)
	}
	for _, vote := range another.Votes() {
		ret.AddVote(vote)
	}
	return ret
}

// UniqueVoterAndBlock consolidate vote set by removing votes from the same voter to same block
// in older epoches.
func (s *VoteSet) UniqueVoterAndBlock() *VoteSet {
	latestVotes := make(map[string]Vote)
	for _, vote := range s.votes {
		key := fmt.Sprintf("%s:%s", vote.ID, vote.Block)
		if prev, ok := latestVotes[key]; ok && prev.Epoch >= vote.Epoch {
			continue
		}
		latestVotes[key] = vote
	}
	ret := NewVoteSet()
	for _, vote := range latestVotes {
		ret.AddVote(vote)
	}
	return ret
}

// UniqueVoter consolidate vote set by removing votes from the same voter in older epoches.
func (s *VoteSet) UniqueVoter() *VoteSet {
	latestVotes := make(map[string]Vote)
	for _, vote := range s.votes {
		key := fmt.Sprintf("%s", vote.ID)
		if prev, ok := latestVotes[key]; ok && prev.Epoch >= vote.Epoch {
			continue
		}
		latestVotes[key] = vote
	}
	ret := NewVoteSet()
	for _, vote := range latestVotes {
		ret.AddVote(vote)
	}
	return ret
}

// FilterByValidators removes votes from non-validators.
func (s *VoteSet) FilterByValidators(validators *AddressSet) *VoteSet {
	ret := NewVoteSet()
	for _, vote := range s.votes {
		if validators.Has(vote.ID) {
			ret.AddVote(vote)
		}
	}
	return ret
}

// VoteDataByID implements sort.Interface for []Vote based on Voter's ID.
type VoteByID []Vote

func (a VoteByID) Len() int           { return len(a) }
func (a VoteByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a VoteByID) Less(i, j int) bool { return bytes.Compare(a[i].ID.Bytes(), a[j].ID.Bytes()) < 0 }
