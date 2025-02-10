package p2p

import (
	"context"

	"github.com/scripttoken/script/common"
	"github.com/scripttoken/script/p2p/types"
)

// MessageHandler interface
type MessageHandler interface {

	// GetChannelIDs returns the list channelIDs that the message handler needs to handle
	GetChannelIDs() []common.ChannelIDEnum

	// ParseMessage parses the raw message bytes
	ParseMessage(peerID string, channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (types.Message, error)

	// EncodeMessage encodes message to bytes
	EncodeMessage(message interface{}) (common.Bytes, error)

	// HandleMessage processes the received message
	HandleMessage(message types.Message) error
}

// Network is a handle to the P2P network
type Network interface {

	// Start is called when the network starts
	Start(ctx context.Context) error

	// Wait blocks until all goroutines have stopped
	Wait()

	// Stop is called when the network stops
	Stop()

	// Broadcast broadcasts the given message to all the neighboring peers
	Broadcast(message types.Message) chan bool

	// BroadcastToNeighbors broadcasts the given message to the neighboring peers
	BroadcastToNeighbors(message types.Message, maxNumPeersToBroadcast int) chan bool

	// Send sends the given message to the peer specified by the peerID
	Send(peerID string, message types.Message) bool

	// Peers return the IDs of all peers
	Peers() []string

	// PeerURLs return the URLs of all peers
	PeerURLs() []string

	// PeerExists indicates if the given peerID is a neighboring peer
	PeerExists(peerID string) bool

	// RegisterMessageHandler registers message handler
	RegisterMessageHandler(messageHandler MessageHandler)

	IsSeedPeer(peerID string) bool

	// ID returns the ID of the network peer
	ID() string
}
