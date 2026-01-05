// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package smtpsender

import (
	"errors"
	"fmt"
	"net/smtp"
)

// SMTPClientConfig holds configuration for the SMTP client.
type SMTPClientConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

type Client struct {
	config SMTPClientConfig
	send   sendFunc
	addr   string
	auth   smtp.Auth
	from   string
}

type sendFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// NewClient creates a new Client.
func NewClient(cfg SMTPClientConfig, from string) (*Client, error) {
	if cfg.Host == "" || cfg.Port == 0 {
		return nil, fmt.Errorf("%w: SMTP host and port are required", ErrSMTPConfig)
	}
	var auth smtp.Auth
	if cfg.Username != "" && cfg.Password != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	return &Client{config: cfg, send: smtp.SendMail, auth: auth, addr: addr, from: from}, nil
}

func (c *Client) From() string {
	return c.from
}

func (c *Client) SendMail(to []string, msg []byte) error {
	err := c.send(c.addr, c.auth, c.from, to, msg)
	if err != nil {
		return fmt.Errorf("%w: failed to send email: %w", ErrSMTPFailedSend, err)
	}

	return nil
}

var (
	ErrSMTPConfig     = errors.New("smtp configuration error")
	ErrSMTPFailedSend = errors.New("smtp failed to send email")
)
