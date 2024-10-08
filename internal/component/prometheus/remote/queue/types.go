package queue

import (
	"fmt"
	"time"

	"github.com/grafana/alloy/internal/component/prometheus/remote/queue/types"
	"github.com/grafana/alloy/syntax/alloytypes"
	"github.com/prometheus/common/version"
	"github.com/prometheus/prometheus/storage"
)

func defaultArgs() Arguments {
	return Arguments{
		TTL: 2 * time.Hour,
		Serialization: Serialization{
			MaxSignalsToBatch: 10_000,
			BatchFrequency:    5 * time.Second,
		},
	}
}

type Arguments struct {
	// TTL is how old a series can be.
	TTL           time.Duration    `alloy:"ttl,attr,optional"`
	Serialization Serialization    `alloy:"serialization,block,optional"`
	Endpoints     []EndpointConfig `alloy:"endpoint,block"`
}

type Serialization struct {
	// The batch size to persist to the file queue.
	MaxSignalsToBatch int `alloy:"max_signals_to_batch,attr,optional"`
	// How often to flush to the file queue if BatchSize isn't met.
	BatchFrequency time.Duration `alloy:"batch_frequency,attr,optional"`
}

type Exports struct {
	Receiver storage.Appendable `alloy:"receiver,attr"`
}

// SetToDefault sets the default
func (rc *Arguments) SetToDefault() {
	*rc = defaultArgs()
}

func defaultEndpointConfig() EndpointConfig {
	return EndpointConfig{
		Timeout:                 30 * time.Second,
		RetryBackoff:            1 * time.Second,
		MaxRetryBackoffAttempts: 0,
		BatchCount:              1_000,
		FlushFrequency:          1 * time.Second,
		QueueCount:              4,
	}
}

func (cc *EndpointConfig) SetToDefault() {
	*cc = defaultEndpointConfig()
}

func (r *Arguments) Validate() error {
	for _, conn := range r.Endpoints {
		if conn.BatchCount <= 0 {
			return fmt.Errorf("batch_count must be greater than 0")
		}
		if conn.FlushFrequency < 1*time.Second {
			return fmt.Errorf("flush_frequency must be greater or equal to 1s, the internal timers resolution is 1s")
		}
	}

	return nil
}

// EndpointConfig is the alloy specific version of ConnectionConfig.
type EndpointConfig struct {
	Name      string        `alloy:",label"`
	URL       string        `alloy:"url,attr"`
	BasicAuth *BasicAuth    `alloy:"basic_auth,block,optional"`
	Timeout   time.Duration `alloy:"write_timeout,attr,optional"`
	// How long to wait between retries.
	RetryBackoff time.Duration `alloy:"retry_backoff,attr,optional"`
	// Maximum number of retries.
	MaxRetryBackoffAttempts uint `alloy:"max_retry_backoff_attempts,attr,optional"`
	// How many series to write at a time.
	BatchCount int `alloy:"batch_count,attr,optional"`
	// How long to wait before sending regardless of batch count.
	FlushFrequency time.Duration `alloy:"flush_frequency,attr,optional"`
	// How many concurrent queues to have.
	QueueCount     uint              `alloy:"queue_count,attr,optional"`
	ExternalLabels map[string]string `alloy:"external_labels,attr,optional"`
}

var UserAgent = fmt.Sprintf("Alloy/%s", version.Version)

func (cc EndpointConfig) ToNativeType() types.ConnectionConfig {
	tcc := types.ConnectionConfig{
		URL:                     cc.URL,
		UserAgent:               UserAgent,
		Timeout:                 cc.Timeout,
		RetryBackoff:            cc.RetryBackoff,
		MaxRetryBackoffAttempts: cc.MaxRetryBackoffAttempts,
		BatchCount:              cc.BatchCount,
		FlushFrequency:          cc.FlushFrequency,
		ExternalLabels:          cc.ExternalLabels,
		Connections:             cc.QueueCount,
	}
	if cc.BasicAuth != nil {
		tcc.BasicAuth = &types.BasicAuth{
			Username: cc.BasicAuth.Username,
			Password: string(cc.BasicAuth.Password),
		}
	}
	return tcc
}

type BasicAuth struct {
	Username string            `alloy:"username,attr,optional"`
	Password alloytypes.Secret `alloy:"password,attr,optional"`
}
