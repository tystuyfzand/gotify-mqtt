package main

import (
	"encoding/json"
	"errors"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gotify/plugin-api"
)

var (
	ErrInvalidAddress = errors.New("invalid broker address")
)

// GetGotifyPluginInfo returns gotify plugin info
func GetGotifyPluginInfo() plugin.Info {
	return plugin.Info{
		Name:       "MQTT",
		ModulePath: "github.com/tystuyfzand/gotify-mqtt",
		Author:     "Tyler Stuyfzand",
		Website:    "https://meow.tf",
	}
}

type Server struct {
	Address string
	Username string
	Password string
	Subscribe []string
}

type Config struct {
	Servers []Server
}

// Plugin is plugin instance
type Plugin struct {
	userCtx    plugin.UserContext
	msgHandler plugin.MessageHandler
	config *Config
	clients []mqtt.Client
	enabled bool
}

// SetMessageHandler implements plugin.Messenger
// Invoked during initialization
func (p *Plugin) SetMessageHandler(h plugin.MessageHandler) {
	p.msgHandler = h
}

// Enable adds users to the context map which maps to a Plugin.
func (p *Plugin) Enable() error {
	p.enabled = true
	return nil
}

// Disable removes users from the context map.
func (p *Plugin) Disable() error {
	p.enabled = false
	p.disconnectClients()
	return nil
}

// DefaultConfig implements plugin.Configurer
// The default configuration will be provided to the user for future editing. Also used for Unmarshaling.
// Invoked whenever an unmarshaling is required.
func (p *Plugin) DefaultConfig() interface{} {
	return &Config{
		Servers: []Server{
			{Address: "127.0.0.1:1883", Subscribe: []string{"*"}},
		},
	}
}

// ValidateAndSetConfig will be called every time the plugin is initialized or the configuration has been changed by the user.
// Plugins should validate the configuration and optionally return an error.
// Parameter is guaranteed to be the same type as the return type of DefaultConfig(), so it is safe to do a hard type assertion here.
//
// "Validation" in this context means to check for conflicting or impossible values, such as a non-URL on a field which should only contain a URL.
// In order to make sure that the plugin instance is always running in a valid state, this method should always accept the result of DefaultConfig()
//
// Invoked on initialization to provide initial configuration. Return nil to accept or return error to indicate that the config is obsolete.
// When the configuration is marked obsolete due to an unmarshaling error or rejection on the plugin side, the plugin is disabled automatically and the user is notified to resolve the config confliction.
// Invoked every time the config update API is called. Check the configuration and return nil to accept or return error to indicate that the config is invalid.
// Return a short and consise error here and, if you have detailed suggestions on how to solve the problem, utilize Displayer to provide more information to the user,
func (p *Plugin) ValidateAndSetConfig(c interface{}) error {
	config := c.(*Config)

	// If listeners are configured, shut them down and start fresh
	if p.clients != nil {
		for _, client := range p.clients {
			if client == nil || !client.IsConnected() {
				continue
			}

			go client.Disconnect(500)
		}
	}

	p.clients = make([]mqtt.Client, len(config.Servers))

	for _, server := range config.Servers {
		if server.Address == "" {
			return ErrInvalidAddress
		}
	}

	p.config = config

	// If enabled already and config was updated, reconnect clients
	if p.enabled {
		return p.connectClients()
	}

	return nil
}

func (p *Plugin) disconnectClients() {
	if p.clients == nil {
		return
	}

	for _, client := range p.clients {
		if client == nil || !client.IsConnected() {
			continue
		}

		go client.Disconnect(500)
	}
}

func (p *Plugin) connectClients() error {
	p.disconnectClients()

	p.clients = make([]mqtt.Client, len(p.config.Servers))

	for i, server := range p.config.Servers {
		client, err := p.newClient(server)

		if err != nil {
			return err
		}

		p.clients[i] = client
	}

	return nil
}

// handleMessage handles mqtt messages from the client by returning a MessageHandler
// Messages are in either JSON format (same as the Gotify API) or simply a string.
func (p *Plugin) handleMessage(client mqtt.Client, message mqtt.Message) {
	payload := message.Payload()

	var outgoingMessage plugin.Message

	if payload[0] == '{' {
		if err := json.Unmarshal(payload, &outgoingMessage); err != nil {
			return
		}
	} else {
		outgoingMessage.Message = string(payload)
	}

	p.msgHandler.SendMessage(outgoingMessage)
}

// newClient creates a new client from the serverConfig
func (p *Plugin) newClient(serverConfig Server) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()

	opts.AddBroker(serverConfig.Address)
	opts.SetClientID("gotify")

	if serverConfig.Username != "" {
		opts.SetUsername(serverConfig.Username)
	}

	if serverConfig.Password != "" {
		opts.SetPassword(serverConfig.Password)
	}

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	for _, topic := range serverConfig.Subscribe {
		client.Subscribe(topic, 0, p.handleMessage)
	}

	return client, nil
}

// NewGotifyPluginInstance creates a plugin instance for a user context.
func NewGotifyPluginInstance(ctx plugin.UserContext) plugin.Plugin {
	return &Plugin{
		userCtx: ctx,
		clients: make([]mqtt.Client, 0),
	}
}

func main() {
	panic("Program must be compiled as a Go plugin")
}
