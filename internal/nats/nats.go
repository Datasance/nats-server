package nats

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"github.com/datasance/nats-server/internal/exec"
	"sync"
)

type Server struct {
	Config *Config
	mu     sync.Mutex // Mutex to ensure that only one server is started at a time
}

type Config struct {
	User        string `json:"User"`
	Password    string `json:"Password"`
	Account     string `json:"Account"`
	Port        int    `json:"Port"`
	Domain      string `json:"Domain"`
	URL         string `json:"URL"`
	URLProtocol string `json:"URLProtocol"`
	TLS         string `json:"TLS"`
	Capath      string `json:"Capath"`
	Tlspath     string `json:"Tlspath"`
	Tlskey      string `json:"Tlskey"`
}

func (s *Server) UpdateServer(config *Config) error {
    s.mu.Lock() // Ensure only one server is started at a time
    defer s.mu.Unlock()

    // Create the new configuration files
    if err := s.createConfigFiles(config); err != nil {
        return err
    }

    log.Printf("NATS server configuration updated successfully.")
    return nil
}

func (s *Server) createConfigFiles(config *Config) error {
	configDir := "./nats-config"
	log.Printf("Creating directory: %s", configDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	accountConfPath := filepath.Join(configDir, "accounts.conf")
	log.Printf("Creating account config file at %s", accountConfPath)
	if err := createAccountConfigFile(accountConfPath, config); err != nil {
		return fmt.Errorf("failed to create account config file: %v", err)
	}

	natsLeafConfPath := filepath.Join(configDir, "nats-leaf.conf")
	log.Printf("Creating NATS leaf config file at %s", natsLeafConfPath)
	if err := createNatsLeafConfigFile(natsLeafConfPath, config); err != nil {
		return fmt.Errorf("failed to create NATS leaf config file: %v", err)
	}

	log.Printf("NATS configuration files updated successfully in %s", configDir)
	return nil
}

func createAccountConfigFile(path string, config *Config) error {
	content := fmt.Sprintf(`accounts {
    %s: {
        users: [{user: %s, password: %s}],
        jetstream: enabled
    }
}`, config.Account, config.User, config.Password)
	return ioutil.WriteFile(path, []byte(content), 0644)
}

func createNatsLeafConfigFile(path string, config *Config) error {
	// Start with the common config content
	content := fmt.Sprintf(`port: %d
jetstream {
    store_dir="./store_leaf"
    domain="%s"
}
leafnodes {
    remotes = [
        {
            urls: ["%s://%s:%s@%s"]
            account: "%s"
`, config.Port, config.Domain, config.URLProtocol, config.User, config.Password, config.URL, config.Account)

	// Check if TLS is enabled and add the TLS block if necessary
	if config.TLS != "" && config.Capath != "" && config.Tlspath != "" && config.Tlskey != "" {
		// TLS configuration provided
		content += fmt.Sprintf(`
            tls: {
                ca_file: %s
                cert_file: %s
                key_file: %s
            }
`, config.Capath, config.Tlspath, config.Tlskey)
	}

	// Closing the remaining part of the configuration
	content += fmt.Sprintf(`
        }
    ]
}
include ./accounts.conf`)

	// Write the content to the file
	return ioutil.WriteFile(path, []byte(content), 0644)
}

func (s *Server) StartServer(config *Config, exitChannel chan error) error {
    // First, update the server configuration and stop any existing server if needed
    if err := s.UpdateServer(config); err != nil {
        return fmt.Errorf("failed to update server before starting: %v", err)
    }

    args := []string{
        fmt.Sprintf("--name=%s-leaf", config.User),
        "-c",
        "nats-config/nats-leaf.conf",
    }

    env := []string{} // Pass any required environment variables here

    go exec.Run(exitChannel, "nats-server", args, env)

    log.Printf("NATS server started successfully with config: %s-leaf", config.User)

    return nil
}
