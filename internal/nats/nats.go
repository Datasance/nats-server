package nats

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/datasance/nats-server/internal/exec"
)

type Server struct {
	Config *Config
	mu     sync.Mutex // Mutex to ensure that only one server is started at a time
}

type Config struct {
	Accounts        []Account    `json:"accounts"`
	NatsServer      NatsServer   `json:"natsServer"`
}

type Account struct {
	AccountName   string `json:"accountName"`
	Users         []User   `json:"users"`
	Jetstream     bool   `json:"jetstream"`
	IsSystem      bool   `json:"isSystem"`
}

type User struct {
	UserName  string `json:"userName"`
	Password  string `json:"password"`
}

type NatsServer struct {
	ServerName  string     `json:"serverName"`
	Port        int        `json:"port"`
	JSDomain    string     `json:"jsDomain"`
    LeafNodes   LeafNode   `json:"leafNodes"`
	TLS         TLS        `json:"tls"`
}

type LeafNode struct {
	Port     int      `json:"port"`
    Remotes  Remote   `json:"remotes"`
}

type Remote struct {
	URLProtocol  string  `json:"urlProtocol"`
	URL          string  `json:"url"`
	User         string  `json:"user"`
	Password     string  `json:"password"`
	Account      string  `json:"account"` 
	TLS          TLS     `json:"tls"`
}

type TLS struct {
	CaCert   string  `json:"caCert"`
	TlsCert  string  `json:"tlsCert"`
	TlsKey   string  `json:"tlsKey"`
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

	if err := s.handleTLSFiles(config, configDir); err != nil {
		return fmt.Errorf("failed to handle TLS files: %v", err)
	}

	accountConfPath := filepath.Join(configDir, "accounts.conf")
	log.Printf("Creating account config file at %s", accountConfPath)
	if err := createAccountConfigFile(accountConfPath, config); err != nil {
		return fmt.Errorf("failed to create account config file: %v", err)
	}

	natsServerConfPath := filepath.Join(configDir, "nats-server.conf")
	log.Printf("Creating NATS server config file at %s", natsServerConfPath)
	if err := createNatsServerConfigFile(natsServerConfPath, config); err != nil {
		return fmt.Errorf("failed to create NATS server config file: %v", err)
	}

	log.Printf("NATS configuration files updated successfully in %s", configDir)
	return nil
}

func (s *Server) handleTLSFiles(config *Config, configDir string) error {
	log.Printf("Handling TLS files in directory: %s", configDir)

	remote := config.NatsServer.LeafNodes.Remotes
	tls := remote.TLS
	serverTls := config.NatsServer.TLS
	serverCertDir := fmt.Sprintf("%s/server-cert", configDir)
	leafCertDir := fmt.Sprintf("%s/leaf-cert", configDir)

	if err := os.MkdirAll(serverCertDir, 0755); err != nil {
		return fmt.Errorf("failed to create server cert directory: %v", err)
	}

	if err := os.MkdirAll(leafCertDir, 0755); err != nil {
		return fmt.Errorf("failed to create leaf cert directory: %v", err)
	}

	if tls.CaCert != "" {
		log.Printf("Processing CaCert for remote: %s", remote.URL)
		caPath := filepath.Join(leafCertDir, "ca.crt")
		if err := decodeCertToFile(tls.CaCert, caPath); err != nil {
			return fmt.Errorf("failed to decode CaCert: %v", err)
		}
	}

	if tls.TlsCert != "" {
		log.Printf("Processing TlsCert for remote: %s", remote.URL)
		tlsCertPath := filepath.Join(leafCertDir, "tls.crt")
		if err := decodeCertToFile(tls.TlsCert, tlsCertPath); err != nil {
			return fmt.Errorf("failed to decode TlsCert: %v", err)
		}
	}

	if tls.TlsKey != "" {
		log.Printf("Processing TlsKey for remote: %s", remote.URL)
		tlsKeyPath := filepath.Join(leafCertDir, "tls.key")
		if err := decodeCertToFile(tls.TlsKey, tlsKeyPath); err != nil {
			return fmt.Errorf("failed to decode TlsKey: %v", err)
		}
	}

	if serverTls.CaCert != "" {
		log.Printf("Processing CaCert for remote")
		caPath := filepath.Join(serverCertDir, "ca.crt")
		if err := decodeCertToFile(tls.CaCert, caPath); err != nil {
			return fmt.Errorf("failed to decode CaCert: %v", err)
		}
	}

	if serverTls.TlsCert != "" {
		log.Printf("Processing TlsCert for remote")
		tlsCertPath := filepath.Join(serverCertDir, "tls.crt")
		if err := decodeCertToFile(tls.TlsCert, tlsCertPath); err != nil {
			return fmt.Errorf("failed to decode TlsCert: %v", err)
		}
	}

	if serverTls.TlsKey != "" {
		log.Printf("Processing TlsKey for server")
		tlsKeyPath := filepath.Join(serverCertDir, "tls.key")
		if err := decodeCertToFile(tls.TlsKey, tlsKeyPath); err != nil {
			return fmt.Errorf("failed to decode TlsKey: %v", err)
		}
	}

	return nil
}


func decodeCertToFile(certString string, outputPath string) error {
	log.Printf("Starting decodeCertToFile for outputPath: %s", outputPath)

	// Decode the base64 data
	decodedData, err := base64.StdEncoding.DecodeString(certString)
	if err != nil {
		log.Fatalf("Failed to decode base64 data: %v", err)
	}

	// Write the decoded data to the file
	if err := ioutil.WriteFile(outputPath, decodedData, 0644); err != nil {
		log.Printf("Failed to write PEM data to file %s: %v", outputPath, err)
		return fmt.Errorf("failed to write PEM data to file: %v", err)
	}

	log.Printf("Successfully wrote PEM file to: %s", outputPath)
	return nil
}

func createAccountConfigFile(path string, config *Config) error {
	var accountsConfig strings.Builder
	var systemAccountName string

	// Start the accounts block
	accountsConfig.WriteString("accounts: {\n")

	// Iterate over all accounts in the config
	for _, account := range config.Accounts {
		accountsConfig.WriteString(fmt.Sprintf("    %s: {\n", account.AccountName))

		// Add users for the account
		accountsConfig.WriteString("        users: [\n")
		for _, user := range account.Users {
			accountsConfig.WriteString(fmt.Sprintf("            {user: %s, password: %s},\n", user.UserName, user.Password))
		}
		accountsConfig.WriteString("        ],\n")

		// Add Jetstream if enabled
		if account.Jetstream {
			accountsConfig.WriteString("        jetstream: enabled\n")
		}

		accountsConfig.WriteString("    },\n")

		// Capture the system account name if marked as system
		if account.IsSystem {
			systemAccountName = account.AccountName
		}
	}

	// Close the accounts block
	accountsConfig.WriteString("}\n")

	// Add the system account if one is defined
	if systemAccountName != "" {
		accountsConfig.WriteString(fmt.Sprintf("system_account: %s\n", systemAccountName))
	}
	
	// Write the configuration to the specified file path
	return ioutil.WriteFile(path, []byte(accountsConfig.String()), 0644)
}

func createNatsServerConfigFile(path string, config *Config) error {
    var content strings.Builder

    natsServer := config.NatsServer

    // Common settings
    content.WriteString(fmt.Sprintf("port: %d\n", natsServer.Port))
    if natsServer.ServerName != "" {
        content.WriteString(fmt.Sprintf("server_name: %s\n", natsServer.ServerName))
    }
    if natsServer.JSDomain != "" {
        content.WriteString(fmt.Sprintf(`jetstream {
	store_dir="./store_leaf"	
    domain: "%s"
}
`, natsServer.JSDomain))
    }

    // Leaf node settings
    content.WriteString("leafnodes {\n")
    leafNode := natsServer.LeafNodes
	serverTLS := natsServer.TLS
    if leafNode.Port > 0 {
        content.WriteString(fmt.Sprintf("    port: %d\n", leafNode.Port))
    }

    // Check if TLS is defined for leafnodes
    if serverTLS.CaCert != "" || serverTLS.TlsCert != "" || serverTLS.TlsKey != "" {
        content.WriteString(`    tls: {
        ca_file: "/nats-config/server-cert/ca.crt"
        cert_file: "/nats-config/server-cert/tls.crt"
        key_file: "/nats-config/server-cert/tls.key"
    }
`)
    }

    // Remotes block
    remote := leafNode.Remotes
    content.WriteString(fmt.Sprintf(`    remotes = [
        {
            urls: ["%s://%s:%s@%s"]
            account: "%s"
`, remote.URLProtocol, remote.User, remote.Password, remote.URL, remote.Account))

    // Check if TLS is defined for remotes
    if remote.TLS.CaCert != "" || remote.TLS.TlsCert != "" || remote.TLS.TlsKey != "" {
        content.WriteString(`            tls: {
                ca_file: "/nats-config/leaf-cert/ca.crt"
                cert_file: "/nats-config/leaf-cert/tls.crt"
                key_file: "/nats-config/leaf-cert/tls.key"
            }
`)
    }
    content.WriteString("        }\n    ]\n")
    content.WriteString("}\n")

    // Include accounts file
    content.WriteString("include ./accounts.conf\n")

    // Write the configuration to the specified file path
    return ioutil.WriteFile(path, []byte(content.String()), 0644)
}


func (s *Server) StartServer(config *Config, exitChannel chan error) error {

	// First, update the server configuration and stop any existing server if needed
	if err := s.UpdateServer(config); err != nil {
		return fmt.Errorf("failed to update server before starting: %v", err)
	}

	args := []string{
		"-c",
		"nats-config/nats-server.conf",
	}

	env := []string{} // Pass any required environment variables here

	go exec.Run(exitChannel, "nats-server", args, env)

	log.Printf("NATS server started successfully")

	return nil
}
