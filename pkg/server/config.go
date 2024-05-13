package server

import (
	"context"
	"crypto/x509/pkix"
	"net"
	"sync"
	"time"
	"fmt"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	common "github.com/spiffe/spire/pkg/common/catalog"
	"github.com/spiffe/spire/pkg/common/health"
	"github.com/spiffe/spire/pkg/common/telemetry"
	loggerv1 "github.com/spiffe/spire/pkg/server/api/logger/v1"
	"github.com/spiffe/spire/pkg/server/authpolicy"
	bundle_client "github.com/spiffe/spire/pkg/server/bundle/client"
	"github.com/spiffe/spire/pkg/server/endpoints"
	"github.com/spiffe/spire/pkg/server/endpoints/bundle"
	"github.com/spiffe/spire/pkg/server/plugin/keymanager"
)

type ConfigListener interface {
	// this method is contraversial, but it would primarly
	// exist to externalize the code that would be within
	// ConfigChanged to validate new configurations could be
	// used prior to using them.
	CheckConfig(config *Config) error

	// The command to use the new configuration.
	ConfigChanged(config *Config) error
}

type Config struct {
	// lock for consistent fine grained updates of the Config.
	mu sync.Mutex

	// the listeners for configuration changes.
	listeners []ConfigListener

	// Configurations for server plugins
	PluginConfigs common.PluginConfigs

	Log loggerv1.Logger

	// LogReopener facilitates handling a signal to rotate log file.
	LogReopener func(context.Context) error

	// If true enables audit logs
	AuditLogEnabled bool

	// Address of SPIRE server
	BindAddress *net.TCPAddr

	// Address of SPIRE Server to be reached locally
	BindLocalAddress net.Addr

	// Directory to store runtime data
	DataDir string

	// Trust domain
	TrustDomain spiffeid.TrustDomain

	Experimental ExperimentalConfig

	// If true enables profiling.
	ProfilingEnabled bool

	// Port used by the pprof web server when ProfilingEnabled == true
	ProfilingPort int

	// Frequency in seconds by which each profile file will be generated.
	ProfilingFreq int

	// Array of profiles names that will be generated on each profiling tick.
	ProfilingNames []string

	// AgentTTL is time-to-live for agent SVIDs
	AgentTTL time.Duration

	// X509SVIDTTL is default time-to-live for X509-SVIDs (overrides SVIDTTL)
	X509SVIDTTL time.Duration

	// JWTSVIDTTL is default time-to-live for SVIDs (overrides SVIDTTL)
	JWTSVIDTTL time.Duration

	// CATTL is the time-to-live for the server CA. This only applies to
	// self-signed CA certificates, otherwise it is up to the upstream CA.
	CATTL time.Duration

	// JWTIssuer is used as the issuer claim in JWT-SVIDs minted by the server.
	// If unset, the JWT-SVID will not have an issuer claim.
	JWTIssuer string

	// CASubject is the subject used in the CA certificate
	CASubject pkix.Name

	// Telemetry provides the configuration for metrics exporting
	Telemetry telemetry.FileConfig

	// HealthChecks provides the configuration for health monitoring
	HealthChecks health.Config

	// CAKeyType is the key type used for the X509 and JWT signing keys
	CAKeyType keymanager.KeyType

	// JWTKeyType is the key type used for JWT signing keys
	JWTKeyType keymanager.KeyType

	// Federation holds the configuration needed to federate with other
	// trust domains.
	Federation FederationConfig

	// RateLimit holds rate limiting configurations.
	RateLimit endpoints.RateLimitConfig

	// CacheReloadInterval controls how often the in-memory entry cache reloads
	CacheReloadInterval time.Duration

	// EventsBasedCache enabled event driven cache reloads
	EventsBasedCache bool

	// PruneEventsOlderThan controls how long events can live before they are pruned
	PruneEventsOlderThan time.Duration

	// AuthPolicyEngineConfig determines the config for authz policy
	AuthOpaPolicyEngineConfig *authpolicy.OpaEngineConfig

	// AdminIDs are a list of fixed IDs that when presented by a caller in an
	// X509-SVID, are granted admin rights.
	AdminIDs []spiffeid.ID

	// Temporary flag to allow disabling the inclusion of serial number in X509 CAs Subject field
	ExcludeSNFromCASubject bool
}

type ExperimentalConfig struct {
}

type FederationConfig struct {
	// BundleEndpoint contains the federation bundle endpoint configuration.
	BundleEndpoint *bundle.EndpointConfig
	// FederatesWith holds the federation configuration for trust domains this
	// server federates with.
	FederatesWith map[spiffeid.TrustDomain]bundle_client.TrustDomainConfig
}


func New(config Config) *Server {
	config.listeners = make([]ConfigListener, 0, 0)
	return &Server{
		config: &config,
	}
}

func (c *Config) AddListener(listener ConfigListener) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, listener)
	return nil
}

func (c *Config) RemoveListener(listener ConfigListener) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	foundIndex := -1
	for index, candidate := range c.listeners {
		if candidate == listener {
			foundIndex = index
			break
		}
	}
	if foundIndex == -1 {
		return fmt.Errorf("ConfigListener %T: %p cannot be removed as a listener to %+v, it is already not listening", listener, listener, c)
	}
	c.listeners = append(c.listeners[:foundIndex], c.listeners[foundIndex+1:]...)
	return nil
}


func (c *Config) checkConfig(proposed *Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, listener := range c.listeners {
		if err := listener.CheckConfig(proposed); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) fireConfigChanged() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, listener := range c.listeners {
		if err := listener.ConfigChanged(c); err != nil {
			return err
		}
	}
	return nil
}

