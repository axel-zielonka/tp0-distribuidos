package common

import (
	"context"
	"net"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

const TIEMPO_ESPERA_REINTENTO = 100

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
	c.createClientSocket()

	c.protocol = NewClientProtocol(c.config.ID, c.conn)

	log.Infof("action: start_sending_bets | result: in_progress")

	err := c.protocol.sendMessageType(c.conn, BET_MESSAGE)
	if err != nil {
		log.Infof("action: start_sending_bets | result: fail | error: %v", err)
		c.closeConnection()
		return
	} else {
		log.Infof("action: start_sending_bets | result: success")
	}

	err = c.protocol.sendBets(c.conn, c.bets, c.config.MaxBatchSize)
	if err != nil {
		log.Infof("action: send_bets | result: fail | error: %v", err)
		c.closeConnection()
		return
	}

	log.Infof("action: send_bets | result: success")

	log.Infof("action: finishing_sending_bets | result: in_progress")

	err = c.protocol.sendMessageType(c.conn, FINISH_MESSAGE)
	if err != nil {
		log.Infof("action: finishing_sending_bets | result: fail | error: %v", err)
		c.closeConnection()
		return
	} else {
		log.Infof("action: finishing_sending_bets | result: success")
	}

	serverResponse, err := c.protocol.receiveMessage()
	if err != nil {
		log.Infof("action: receive_message | result: fail | error: %v", err)
		c.closeConnection()
		return
	}

	response, err := c.protocol.parseResponseFromServer(serverResponse)

	if err != nil {
		log.Infof("action: receive_message | result: fail | error: %v", err)
		c.closeConnection()
		return
	}

	c.handleServerResponse(response)

	c.closeConnection()
	for {
		select {
		case <-ctx.Done():
			c.closeConnection()
			log.Infof("action: shutdown | result: succes | client_id: %v", c.config.ID)
			return
		default:
			log.Infof("action: consulta_ganadores | result: in_progress")
			_ = c.createClientSocket()
			c.protocol.changeSocket(c.conn)
			ganadores, err := c.protocol.askForResults(c.conn, ASK_FOR_RESULT_MESSAGE)
			if ganadores < 0 {
				c.closeConnection()
				time.After(time.Duration(TIEMPO_ESPERA_REINTENTO * time.Millisecond))
			}
			if err == nil {
				log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", ganadores)
				c.closeConnection()
				return
			} else {
				c.closeConnection()
			}
		}
	}
}

func (c *Client) handleServerResponse(response *ServerResponse) {
	if response.betCount != len(c.bets) {
		log.Infof("action: apuesta_almacenada | result: fail | cantidad: %d | error: %v", response.betCount, response.Message)
	} else {
		log.Infof("action: apuesta_almacenada | result: success | cantidad: %d", response.betCount)
	}
}
