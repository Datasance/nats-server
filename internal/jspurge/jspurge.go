package jspurge

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	jsAccountPurgeSubjectT = "$JS.API.ACCOUNT.PURGE.%s"
	purgeRequestTimeout    = 10 * time.Second
)

// APIResponse is the standard JetStream API response (type + optional error).
type APIResponse struct {
	Type  string    `json:"type"`
	Error *APIError `json:"error,omitempty"`
}

// APIError is the error field in APIResponse.
type APIError struct {
	Code        int    `json:"code"`
	Description string `json:"description,omitempty"`
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Description
}

// JSApiAccountPurgeResponse is the response for account purge (includes initiated).
type JSApiAccountPurgeResponse struct {
	APIResponse
	Initiated bool `json:"initiated,omitempty"`
}

// AccountsFromJWTDir lists account names from the JWT resolver directory.
// It reads files matching *.jwt and returns the filename without .jwt.
// Files ending with .delete (e.g. *.jwt.delete when allow_delete is true) are skipped.
func AccountsFromJWTDir(jwtDir string) ([]string, error) {
	entries, err := readDirNames(jwtDir)
	if err != nil {
		return nil, err
	}
	var accounts []string
	for _, name := range entries {
		if !strings.HasSuffix(name, ".jwt") {
			continue
		}
		if strings.HasSuffix(name, ".delete") {
			continue
		}
		account := strings.TrimSuffix(name, ".jwt")
		if account != "" {
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}

// AccountsFromJetStreamStore lists account IDs that have JetStream data on disk.
// It returns the names of immediate subdirectories under storeDir/jetstream.
// If storeDir or storeDir/jetstream does not exist or is not a directory, returns nil, nil.
func AccountsFromJetStreamStore(storeDir string) ([]string, error) {
	jetstreamDir := filepath.Join(storeDir, "jetstream")
	entries, err := os.ReadDir(jetstreamDir)
	if err != nil {
		return nil, nil
	}
	var accounts []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "" || name == "." || name == ".." {
			continue
		}
		accounts = append(accounts, name)
	}
	return accounts, nil
}

func readDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

// PurgeAccount calls the JetStream Account Purge API for the given account using system credentials.
// Subject: $JS.API.ACCOUNT.PURGE.{accountName}, body: {}. Returns nil if the server reports success (initiated: true or no error).
func PurgeAccount(ctx context.Context, natsURL, credsPath, accountName string) error {
	var opts []nats.Option
	if credsPath != "" {
		opts = append(opts, nats.UserCredentials(credsPath))
	}
	nc, err := nats.Connect(natsURL, opts...)
	if err != nil {
		return err
	}
	defer nc.Close()

	subject := strings.Replace(jsAccountPurgeSubjectT, "%s", accountName, 1)
	req := []byte("{}")
	reqCtx, cancel := context.WithTimeout(ctx, purgeRequestTimeout)
	defer cancel()
	msg, err := nc.RequestWithContext(reqCtx, subject, req)
	if err != nil {
		return err
	}
	var resp JSApiAccountPurgeResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	_ = resp.Initiated // server may omit initiated; treat as success for idempotency
	return nil
}

// ToPurge returns account names that are in accountsWithJS but not in currentResolver.
// These are the accounts that should be purged (removed from resolver but still have JS data).
func ToPurge(accountsWithJS, currentResolver []string) []string {
	resolverSet := make(map[string]struct{}, len(currentResolver))
	for _, a := range currentResolver {
		resolverSet[a] = struct{}{}
	}
	var out []string
	for _, a := range accountsWithJS {
		if _, ok := resolverSet[a]; !ok {
			out = append(out, a)
		}
	}
	return out
}
