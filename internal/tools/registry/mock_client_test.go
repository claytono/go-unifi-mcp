package registry

import "github.com/filipowm/go-unifi/unifi"

// mockClient embeds a nil pointer to satisfy the unifi.Client interface.
// All methods will panic if called, which is fine because our test uses
// a custom validator that fails before any methods are invoked.
type mockClient struct {
	unifi.Client
}
