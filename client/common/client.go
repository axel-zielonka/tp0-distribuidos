package common

import (
	"context"
	"net"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
	MaxBatchSize  int
}

// Client Entity that encapsulates how
type Client struct {
	config  	ClientConfig
	conn    	net.Conn
	bets 		[]BetInfo
	protocol 	ClientProtocol
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}

	bets, err := loadBetsFromFile(client.config.ID)

	if err != nil {
		log.Criticalf("action: load_bet_data | result: fail | client_id: %v | error: %v", config.ID, err)
	}

	client.bets = bets

	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func(c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
	}
	c.conn = conn
	return nil
}

// closeConnection Close connection with logging
func(c *Client) closeConnection() {
	if c.conn != nil {
		c.conn.Close()
		log.Infof("action: close_connection | result: success | client_id: %v", c.config.ID)
		c.conn = nil
	}
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop(ctx context.Context) {
	select {
	case <-ctx.Done():
		log.Infof("action: shutdown | result: success | client_id: %v", c.config.ID)
		if c.conn != nil {
			c.closeConnection()
		}
		return
	default:
	}

	c.createClientSocket()

	c.protocol = NewClientProtocol(c.config.ID, c.conn)

	err := c.protocol.sendBets(c.conn, c.bets, c.config.MaxBatchSize)
	if err != nil {
		log.Infof("action: send_bets | result: fail | error: %v", err)
		c.closeConnection()
		return
	}

	log.Infof("action: send_bets | result: success")

	serverResponse, err := c.protocol.receiveMessage()
	if err != nil {
		log.Infof("action: receive_message | result: fail | error: %v", err)
		c.closeConnection()
		return
	}

	c.closeConnection()

	response, err := c.protocol.parseResponseFromServer(serverResponse)

	if err != nil {
		log.Infof("action: receive_message | result: fail | error: %v", err)
		c.closeConnection()
		return
	}

	c.handleServerResponse(response)
	

	select {
	case <-ctx.Done():
		c.closeConnection()
		log.Infof("action: shutdown | result: succes | client_id: %v", c.config.ID)
		return
	case <-time.After(c.config.LoopPeriod):
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

func (c *Client) handleServerResponse(response *ServerResponse) {
	if response.betCount != len(c.bets) {
		log.Infof("action: apuesta_almacenada | result: fail | cantidad: %d | error: %v", response.betCount, response.Message)
	} else {
		log.Infof("action: apuesta_almacenada | result: success | cantidad: %d", response.betCount)
	}
}
