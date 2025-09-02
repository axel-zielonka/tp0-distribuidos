package common

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

type BetInfo struct {
	Agency 		string
	Name 		string
	Surname 	string
	Document 	string
	Birthdate 	string
	Number 		string
}

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
	betInfo BetInfo
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	
	if err := client.loadBetDataFromEnv(); err != nil {
		log.Criticalf("action: load_bet_data | result: fail | client_id: %v | error: %v", config.ID, err)
	}

	return client
}

// loadBetDataFromEnv reads environment variables and creates BetInfo struct
func (c* Client) loadBetDataFromEnv() error {
	c.betInfo.Agency = c.config.ID
	c.betInfo.Name = os.Getenv("NAME")
	c.betInfo.Surname = os.Getenv("SURNAME")
	c.betInfo.Document = os.Getenv("DOCUMENT")
	c.betInfo.Birthdate = os.Getenv("BIRTHDATE")

	numberStr := os.Getenv("NUMBER")
	if numberStr == "" {
		return fmt.Errorf("NUMBER environment variable is required")
	}
	
	c.betInfo.Number = numberStr

	if c.betInfo.Name == "" || c.betInfo.Surname == "" || c.betInfo.Document == "" || c.betInfo.Birthdate == "" {
		return fmt.Errorf("All bet data fields are required")
	}

	return nil
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// closeConnection Close connection with logging
func (c *Client) closeConnection() {
	if c.conn != nil {
		c.conn.Close()
		log.Infof("action: close_connection | result: success | client_id: %v", c.config.ID)
		c.conn = nil
	}
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop(ctx context.Context) {
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		// Check if shutdown was requested
		select {
		case <-ctx.Done():
			log.Infof("action: shutdown | result: success | client_id: %v | message_id: %v", c.config.ID, msgID)
			if c.conn != nil {
				c.closeConnection()
			}
			return
		default:
			// continue
		}

		// Create the connection to the server in every loop iteration. Send an
		c.createClientSocket()

		// TODO: Modify the send to avoid short-write
		fmt.Fprintf(
			c.conn,
			"[CLIENT %v] Message N°%v\n",
			c.config.ID,
			msgID,
		)
		msg, err := bufio.NewReader(c.conn).ReadString('\n')
		c.closeConnection()

		if err != nil {
			log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}

		log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
			c.config.ID,
			msg,
		)

		// Wait a time between sending one message and the next one
		// This sleep is now interruptible by shutdown signal
		select {
		case <-ctx.Done():
			log.Infof("action: shutdown | result: success | client_id: %v", c.config.ID)
			return
		case <-time.After(c.config.LoopPeriod):
			// Continue to next iteration
		}
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
