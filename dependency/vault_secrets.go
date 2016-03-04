package dependency

import (
	"fmt"
	"log"
	"sort"
	"time"
)

// VaultSecrets is the dependency to list secrets in Vault.
type VaultSecrets struct {
	Path string
}

// Fetch queries the Vault API
func (d *VaultSecrets) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	if opts == nil {
		opts = &QueryOptions{}
	}

	log.Printf("[DEBUG] (%s) querying vault with %+v", d.Display(), opts)

	// If this is not the first query and we have a lease duration, sleep until we
	// try to renew.
	if opts.WaitIndex != 0 {
		log.Printf("[DEBUG] (%s) pretending to long-poll", d.Display())
		time.Sleep(sleepTime)
	}

	// Grab the vault client
	vault, err := clients.Vault()
	if err != nil {
		return nil, nil, fmt.Errorf("vault secrets: %s", err)
	}

	// Get the list as a secret
	vaultSecret, err := vault.Logical().List(d.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("error listing secrets from vault: %s", err)
	}

	// The secret could be nil (maybe it does not exist yet). This is not an error
	// to Vault, but it is an error to Consul Template, so return an error
	// instead.
	if vaultSecret == nil {
		return nil, nil, fmt.Errorf("no secrets exist at path %q", d.Display())
	}

	// If the data is nil, return an empty list of strings.
	if vaultSecret.Data == nil {
		return respWithMetadata(make([]string, 0))
	}

	// If there are no keys at that path, return the empty list.
	keys, ok := vaultSecret.Data["keys"]
	if !ok {
		return respWithMetadata(make([]string, 0))
	}

	// Convert the interface into a list of interfaces.
	list, ok := keys.([]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("vault returned an unexpected payload for %q", d.Display())
	}

	// Pull each item out of the list and safely cast to a string.
	result := make([]string, len(list))
	for i, v := range list {
		typed, ok := v.(string)
		if !ok {
			return nil, nil, fmt.Errorf("vault returned a non-string when listing secrets for %q", d.Display())
		}
		result[i] = typed
	}
	sort.Strings(result)

	log.Printf("[DEBUG] (%s) vault listed %d secrets(s)", d.Display(), len(result))

	return respWithMetadata(result)
}

// CanShare returns if this dependency is shareable.
func (d *VaultSecrets) CanShare() bool {
	return false
}

// HashCode returns the hash code for this dependency.
func (d *VaultSecrets) HashCode() string {
	return fmt.Sprintf("VaultSecrets|%s", d.Path)
}

// Display returns a string that should be displayed to the user in output (for
// example).
func (d *VaultSecrets) Display() string {
	return fmt.Sprintf(`"secrets(%s)"`, d.Path)
}

// ParseVaultSecrets creates a new datacenter dependency.
func ParseVaultSecrets(s string) (*VaultSecrets, error) {
	// Ensure a trailing slash, always.
	if len(s) == 0 {
		s = "/"
	}
	if s[len(s)-1] != '/' {
		s = fmt.Sprintf("%s/", s)
	}

	return &VaultSecrets{Path: s}, nil
}
