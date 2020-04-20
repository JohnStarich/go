package scutil

type Reach string

// docs lifted from the 'scutil' man page

const (
	// NotReachable is when the specified nodename/address cannot be reached using
	// the current network configuration.
	NotReachable Reach = "Not Reachable"
	// Reachable is when the specified nodename/address can be reached using the
	// current network configuration.
	Reachable Reach = "Reachable"
	// TransientReachable is when the specified nodename/address can be reached
	// via a transient (e.g. PPP) connection.
	TransientReachable Reach = "Transient Connection"
	// ConnectionRequired is when the specified nodename/address can be reached
	// using the current network configuration but a connection must first be established.
	// As an example, this status would be returned for a dialup connection that
	// was not currently active but could handle network traffic for the target system.
	ConnectionRequired Reach = "Connection Required"
	// ConnectionAutomatic is when the specified nodename/address can be reached
	// using the current network configuration but a connection must first be
	// established. Any traffic directed to the specified name/address will
	// initiate the connection.
	ConnectionAutomatic Reach = "Connection Automatic"
	// LocalAddress is when the specified nodename/address is one associated
	// with a network interface on the system.
	LocalAddress Reach = "Local Address"
	// DirectlyReachableAddress is when the network traffic to the specified
	// nodename/address will not go through a gateway but is routed directly to
	// one of the interfaces on the system.
	DirectlyReachableAddress Reach = "Directly Reachable Address"
)
