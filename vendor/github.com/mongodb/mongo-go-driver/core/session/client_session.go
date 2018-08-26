package session

import (
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/uuid"
)

// ErrSessionEnded is returned when a client session is used after a call to endSession().
var ErrSessionEnded = errors.New("ended session was used")

// Client is a session for clients to run commands.
type Client struct {
	ClientID      uuid.UUID
	ClusterTime   *bson.Document
	Consistent    bool // causal consistency
	OperationTime *bson.Timestamp
	SessionID     *bson.Document
	SessionType   Type
	Terminated    bool

	pool          *Pool
	serverSession *Server
}

func getClusterTime(clusterTime *bson.Document) (uint32, uint32) {
	if clusterTime == nil {
		return 0, 0
	}

	clusterTimeVal, err := clusterTime.LookupErr("$clusterTime")
	if err != nil {
		return 0, 0
	}

	timestampVal, err := clusterTimeVal.MutableDocument().LookupErr("clusterTime")
	if err != nil {
		return 0, 0
	}

	return timestampVal.Timestamp()
}

// MaxClusterTime compares 2 clusterTime documents and returns the document representing the highest cluster time.
func MaxClusterTime(ct1 *bson.Document, ct2 *bson.Document) *bson.Document {
	epoch1, ord1 := getClusterTime(ct1)
	epoch2, ord2 := getClusterTime(ct2)

	if epoch1 > epoch2 {
		return ct1
	} else if epoch1 < epoch2 {
		return ct2
	} else if ord1 > ord2 {
		return ct1
	} else if ord1 < ord2 {
		return ct2
	}

	return ct1
}

// NewClientSession creates a Client.
func NewClientSession(pool *Pool, clientID uuid.UUID, sessionType Type, opts ...ClientOptioner) (*Client, error) {
	c := &Client{
		Consistent:  true, // causal consistency defaults to true
		ClientID:    clientID,
		SessionType: sessionType,
		pool:        pool,
	}

	var err error
	for _, opt := range opts {
		err = opt.Option(c)
		if err != nil {
			return nil, err
		}
	}

	servSess, err := pool.GetSession()
	if err != nil {
		return nil, err
	}

	c.SessionID = servSess.SessionID
	c.serverSession = servSess

	return c, nil
}

// AdvanceClusterTime updates the session's cluster time.
func (c *Client) AdvanceClusterTime(clusterTime *bson.Document) error {
	if c.Terminated {
		return ErrSessionEnded
	}
	c.ClusterTime = MaxClusterTime(c.ClusterTime, clusterTime)
	return nil
}

// AdvanceOperationTime updates the session's operation time.
func (c *Client) AdvanceOperationTime(opTime *bson.Timestamp) error {
	if c.Terminated {
		return ErrSessionEnded
	}

	if c.OperationTime == nil {
		c.OperationTime = opTime
		return nil
	}

	if opTime.T > c.OperationTime.T {
		c.OperationTime = opTime
	} else if (opTime.T == c.OperationTime.T) && (opTime.I > c.OperationTime.I) {
		c.OperationTime = opTime
	}

	return nil
}

// UpdateUseTime updates the session's last used time.
// Must be called whenver this session is used to send a command to the server.
func (c *Client) UpdateUseTime() error {
	if c.Terminated {
		return ErrSessionEnded
	}
	c.serverSession.updateUseTime()
	return nil
}

// EndSession ends the session.
func (c *Client) EndSession() {
	if c.Terminated {
		return
	}

	c.Terminated = true
	c.pool.ReturnSession(c.serverSession)
	return
}

// Type describes the type of the session
type Type uint8

// These constants are the valid types for a client session.
const (
	Explicit Type = iota
	Implicit
)
