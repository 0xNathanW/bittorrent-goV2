package client

import (
	"errors"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/0xNathanW/bittorrent-go/p2p"
	"github.com/0xNathanW/bittorrent-go/p2p/message"
	"github.com/0xNathanW/bittorrent-go/torrent"
	"github.com/0xNathanW/bittorrent-go/tracker"
	"github.com/0xNathanW/bittorrent-go/ui"
)

// Client is the highest level of the application.
type Client struct {
	ID       [20]byte // The client's unique ID.
	Torrent  *torrent.Torrent
	Peers    *Peers
	Tracker  *tracker.Tracker
	BitField message.Bitfield
	UI       *ui.UI
	Seed     *sync.Cond // Used to signal when to start seeding.
}

type Peers struct {
	sync.RWMutex
	active   map[[20]byte]*p2p.Peer
	inactive []*net.TCPAddr
}

// Create a new client instance.
// Contains all information required to start download.
func NewClient(path string) (*Client, error) {

	// Unpack and parse torrent file.
	torrent, err := torrent.NewTorrent(path)
	if err != nil {
		return nil, err
	}

	client := &Client{ // Client instance.
		ID:      idGenerator(),
		Torrent: torrent,
	}

	// Generate empty bitfield.
	numPieces := len(torrent.Pieces)
	if numPieces%8 == 0 {
		client.BitField = make(message.Bitfield, numPieces/8)
	} else {
		client.BitField = make(message.Bitfield, numPieces/8+1)
	}

	// Setup tracker.
	tracker, err := tracker.NewTracker(torrent.Announce, torrent.AnnounceList)
	if err != nil {
		return nil, err
	}
	tracker.InitParams(torrent.InfoHash, client.ID, torrent.Size)
	client.Tracker = tracker

	if err = client.GetPeers(); err != nil {
		return nil, err
	}

	ui, err := ui.NewUI(torrent, client.Peers)
	if err != nil {
		return nil, err
	}
	client.UI = ui

	return client, nil
}

// Generate a new client ID.
func idGenerator() [20]byte {
	rand.Seed(time.Now().UnixNano())
	var id [20]byte
	rand.Read(id[:])
	return id
}

// Client retrieves and parses peers from tracker.
func (c *Client) GetPeers() error {

	peersString, err := c.Tracker.RequestPeers()
	if err != nil {
		return err
	}

	peers, inactive := p2p.ParsePeers(peersString, len(c.BitField))
	if len(peers) == 0 {
		return errors.New("No peers found.")
	}

	c.Peers = &Peers{
		active:   peers,
		inactive: inactive,
	}
	c.Peers.Lock()

	return nil
}
