package consensus

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/common/util"
	"github.com/scripttoken/script/core"
	"github.com/scripttoken/script/crypto/bls"
)

const (
	maxLogNeighbors uint32 = 3 // Estimated number of neighbors during gossip = 2**3 = 8
	maxRound               = 10
)

type LightningEngine struct {
	logger *log.Entry

	engine  *ConsensusEngine
	privKey *bls.SecretKey

	// State for current voting
	block       common.Hash
	round       uint32
	currVote    *core.AggregatedVotes
	nextVote    *core.AggregatedVotes
	gcp         *core.LightningCandidatePool
	gcpHash     common.Hash
	signerIndex int // Signer's index in current gcp

	incoming chan *core.AggregatedVotes
	mu       *sync.Mutex
}

func NewLightningEngine(c *ConsensusEngine, privateKey *bls.SecretKey) *LightningEngine {
	return &LightningEngine{
		logger:  util.GetLoggerForModule("lightning"),
		engine:  c,
		privKey: privateKey,

		incoming: make(chan *core.AggregatedVotes, viper.GetInt(common.CfgConsensusMessageQueueSize)),
		mu:       &sync.Mutex{},
	}
}

func (g *LightningEngine) isLightning() bool {
	return g.signerIndex >= 0
}

func (g *LightningEngine) StartNewBlock(block common.Hash) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.block = block
	g.nextVote = nil
	g.currVote = nil
	g.round = 1

	gcp, err := g.engine.GetLedger().GetLightningCandidatePool(block)
	if err != nil {
		// Should not happen
		g.logger.Panic(err)
	}
	g.gcp = gcp
	g.gcpHash = gcp.Hash()
	g.signerIndex = gcp.WithStake().Index(g.privKey.PublicKey())

	g.logger.WithFields(log.Fields{
		"block":       block.Hex(),
		"gcp":         g.gcpHash.Hex(),
		"signerIndex": g.signerIndex,
	}).Debug("Starting new block")

	if g.isLightning() {
		g.nextVote = core.NewAggregateVotes(block, gcp)
		g.nextVote.Sign(g.privKey, g.signerIndex)
		g.currVote = g.nextVote.Copy()
	} else {
		g.nextVote = nil
		g.currVote = nil
	}

}

func (g *LightningEngine) StartNewRound() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.round < maxRound {
		g.round++
		if g.nextVote != nil {
			g.currVote = g.nextVote.Copy()
		}
	}
}

func (g *LightningEngine) GetVoteToBroadcast() *core.AggregatedVotes {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.currVote
}

func (g *LightningEngine) GetBestVote() *core.AggregatedVotes {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.nextVote
}

func (g *LightningEngine) Start(ctx context.Context) {
	go g.mainLoop(ctx)
}

func (g *LightningEngine) mainLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case vote, ok := <-g.incoming:
			if ok {
				g.processVote(vote)
			}
		}
	}
}

func (g *LightningEngine) processVote(vote *core.AggregatedVotes) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.validateVote(vote) {
		return
	}

	if g.nextVote == nil {
		g.nextVote = vote
		return
	}

	var candidate *core.AggregatedVotes
	var err error
	if !g.isLightning() && viper.GetBool(common.CfgConsensusPassThroughLightningVote) {
		candidate, err = g.nextVote.Pick(vote)
		if err != nil {
			g.logger.WithFields(log.Fields{
				"g.block":               g.block.Hex(),
				"g.round":               g.round,
				"vote.block":            vote.Block.Hex(),
				"vote.Mutiplies":        vote.Multiplies,
				"vote.GCP":              vote.Gcp.Hex(),
				"g.nextVote.Multiplies": g.nextVote.Multiplies,
				"g.nextVote.GCP":        g.nextVote.Gcp.Hex(),
				"g.nextVote.Block":      g.nextVote.Block.Hex(),
				"error":                 err.Error(),
			}).Debug("Failed to pick lightning vote")
		}
		if candidate == g.nextVote {
			// Incoming vote is not better than the current nextVote.
			g.logger.WithFields(log.Fields{
				"vote.block":     vote.Block.Hex(),
				"vote.Mutiplies": vote.Multiplies,
			}).Debug("Skipping vote: not better")
			return
		}
	} else {
		candidate, err = g.nextVote.Merge(vote)
		if err != nil {
			g.logger.WithFields(log.Fields{
				"g.block":               g.block.Hex(),
				"g.round":               g.round,
				"vote.block":            vote.Block.Hex(),
				"vote.Mutiplies":        vote.Multiplies,
				"vote.GCP":              vote.Gcp.Hex(),
				"g.nextVote.Multiplies": g.nextVote.Multiplies,
				"g.nextVote.GCP":        g.nextVote.Gcp.Hex(),
				"g.nextVote.Block":      g.nextVote.Block.Hex(),
				"error":                 err.Error(),
			}).Debug("Failed to merge lightning vote")
		}
		if candidate == nil {
			// Incoming vote is subset of the current nextVote.
			g.logger.WithFields(log.Fields{
				"vote.block":     vote.Block.Hex(),
				"vote.Mutiplies": vote.Multiplies,
			}).Debug("Skipping vote: no new index")
			return
		}
	}

	if !g.checkMultipliesForRound(candidate, g.round+1) {
		g.logger.WithFields(log.Fields{
			"local.block":           g.block.Hex(),
			"local.round":           g.round,
			"vote.block":            vote.Block.Hex(),
			"vote.Mutiplies":        vote.Multiplies,
			"local.vote.Multiplies": g.nextVote.Multiplies,
		}).Debug("Skipping vote: candidate vote overflows")
		return
	}

	g.nextVote = candidate

	g.logger.WithFields(log.Fields{
		"local.block":           g.block.Hex(),
		"local.round":           g.round,
		"local.vote.Multiplies": g.nextVote.Multiplies,
	}).Debug("New lightning vote")

    g.logger.Debug("TR-10000");
}

func (g *LightningEngine) HandleVote(vote *core.AggregatedVotes) {
	select {
	case g.incoming <- vote:
		return
	default:
		g.logger.Debug("LightningEngine queue is full, discarding vote: %v", vote)
	}
}

func (g *LightningEngine) validateVote(vote *core.AggregatedVotes) (res bool) {
	if g.block.IsEmpty() {
		g.logger.WithFields(log.Fields{
			"local.block":    g.block.Hex(),
			"local.round":    g.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Debug("Ignoring lightning vote: local not ready")
		return
	}
	if vote.Block != g.block {
		g.logger.WithFields(log.Fields{
			"local.block":    g.block.Hex(),
			"local.round":    g.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Debug("Ignoring lightning vote: block hash does not match with local candidate")
		return
	}
	if vote.Gcp != g.gcpHash {
		g.logger.WithFields(log.Fields{
			"local.block":    g.block.Hex(),
			"local.round":    g.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
			"vote.gcp":       vote.Gcp.Hex(),
			"local.gcp":      g.gcpHash.Hex(),
		}).Debug("Ignoring lightning vote: gcp hash does not match with local value")
		return
	}
	if !g.checkMultipliesForRound(vote, g.round) {
		g.logger.WithFields(log.Fields{
			"local.block":    g.block.Hex(),
			"local.round":    g.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
			"vote.gcp":       vote.Gcp.Hex(),
			"local.gcp":      g.gcpHash.Hex(),
		}).Debug("Ignoring lightning vote: mutiplies exceed limit for round")
		return
	}
	if result := vote.Validate(g.gcp); result.IsError() {
		g.logger.WithFields(log.Fields{
			"local.block":    g.block.Hex(),
			"local.round":    g.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
			"vote.gcp":       vote.Gcp.Hex(),
			"local.gcp":      g.gcpHash.Hex(),
			"error":          result.Message,
		}).Debug("Ignoring lightning vote: invalid vote")
		return
	}
	res = true
	return
}

func (g *LightningEngine) checkMultipliesForRound(vote *core.AggregatedVotes, k uint32) bool {
	for _, m := range vote.Multiplies {
		if m > g.maxMultiply(k) {
			return false
		}
	}
	return true
}

func (g *LightningEngine) maxMultiply(k uint32) uint32 {
	return 1 << (k * maxLogNeighbors)
}
