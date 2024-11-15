package server

import (
	"context"
	"crypto/x509/pkix"
	"net"
	"time"

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

type Config struct {
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

	// SQLTransactionTimeout controls how long to wait for an event before giving up
	SQLTransactionTimeout time.Duration

	// AuthPolicyEngineConfig determines the config for authz policy
	AuthOpaPolicyEngineConfig *authpolicy.OpaEngineConfig

	// AdminIDs are a list of fixed IDs that when presented by a caller in an
	// X509-SVID, are granted admin rights.
	AdminIDs []spiffeid.ID

	// UseLegacyDownstreamX509CATTL, if true, the downstream X509CAs will use
	// the legacy TTL calculation (e.g. prefer downstream workload entry TTL,
	// then fall back to the default workload X509-SVID TTL) v.s. the new TTL
	// calculation (prefer the TTL passed by the downstream caller, then fall
	// back to the default X509 CA TTL).
	UseLegacyDownstreamX509CATTL bool

	// load this server, validate configurations, and then shutdown
	ValidateOnly bool

	// notes about the configuration
	ValidationNotes []string

	// if the configuration is usable
	ValidationInError bool
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
	return &Server{
		config: config,
	}
}
