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
}

// Client Entity that encapsulates how
type Client struct {
	config  ClientConfig
	conn    net.Conn
	betInfo BetInfo
	protocol ClientProtocol
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}

	betInfo, err := loadBetDataFromEnv(client.config.ID)

	if err != nil {
		log.Criticalf("action: load_bet_data | result: fail | client_id: %v | error: %v", config.ID, err)
	}

	client.betInfo = betInfo

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

	response, err := c.protocol.SendBet(c.betInfo)
	if err != nil {
		log.Errorf("action: send_bet | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}

	c.handleServerResponse(response)

	c.closeConnection()

	select {
	case <-ctx.Done():
		log.Infof("action: shutdown | result: succes | client_id: %v", c.config.ID)
		return
	case <-time.After(c.config.LoopPeriod):
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}

func (c *Client) handleServerResponse(response *ServerResponse) {
	if response.Status == "SUCCESS" {
		log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s", c.betInfo.Document, c.betInfo.Number)
	} else {
		log.Errorf("action: apuesta_enviada | result: success | dni: %s | numero: %s | error: %s", c.betInfo.Document, c.betInfo.Number, response.Message)
	}
}
