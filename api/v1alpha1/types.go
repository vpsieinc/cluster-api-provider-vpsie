package v1alpha1

import (
	"strings"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type VpsieResourceStatus string

type VpsieInstanceStatus string

var (
	InstanceStatusActive = VpsieInstanceStatus("1")

	InstanceStatusInActive = VpsieInstanceStatus("0")

	InstanceStatusPending = VpsieInstanceStatus("pending")

	InstanceStatusSuspended = VpsieInstanceStatus("suspended")

	InstanceStatusLocked = VpsieInstanceStatus("locked")

	InstanceStatusTerminated = VpsieInstanceStatus("terminated")

	InstanceStatusDeleted = VpsieInstanceStatus("deleted")
)

type VpsieMachineTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the specification of the desired behavior of the machine.
	Spec VpsieMachineSpec `json:"spec"`
}

// VpsieResourceReference is a reference to a Vpsie resource.
type VpsieResourceReference struct {
	// ID of Vpsie resource
	// +optional
	ID string `json:"id,omitempty"`

	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	Status string `json:"status,omitempty"`
}

// VpsieNetworkResource encapsulates Vpsie networking resources.
type VpsieNetworkResource struct {
	// APIServerLoadbalancersRef is the id of apiserver loadbalancers.
	// +optional
	APIServerLoadbalancersRef VpsieResourceReference `json:"apiServerLoadbalancersRef,omitempty"`
}

// NetworkSpec encapsulates all things related to Vpsie network.
type NetworkSpec struct {
	// VPC configuration.
	// +optional
	VPC VPCSpec `json:"vpc,omitempty"`

	// LoadBalancer configuration
	APIServerLoadbalancers LoadBalancer `json:"apiServerLoadbalancers,omitempty"`
}

type AdditionalStorage struct {
	DiskFormat  string `json:"diskFormat,omitempty"`
	Size        string `json:"size,omitempty"`
	Name        string `json:"name,omitempty"`
	StorageType string `json:"storageType,omitempty"`
	IsAutomatic int    `json:"isAutomatic,omitempty"`
}

type LoadBalancer struct {
	// The Vpsie load balancer UUID. If omitted, a new load balancer will be created.
	// +optional
	ID string `json:"id,omitempty"`

	// It must be either "round_robin" or "least_connections". The default value is "round_robin".
	// +optional
	// +kubebuilder:validation:Enum=round_robin;least_connections
	Algorithm string `json:"algorithm,omitempty"`

	// +optional
	DcIdentifier string `json:"dcIdentifier,omitempty"`

	// +optional
	LbName string `json:"lbName,omitempty"`

	// +optional
	RedirectHTTP int `json:"redirectHTTP,omitempty"`

	// +optional
	Rules []Rule `json:"rules,omitempty"`

	// +optional
	Domains []Domain `json:"domains,omitempty"`

	// +optional
	HealthCheck VpsieLoadBalancerHealthCheck `json:"healthCheck,omitempty"`

	// +optional
	ResourceIdentifier string `json:"resourceIdentifier,omitempty"`
}

type Rule struct {
	// +optional
	Scheme string `json:"scheme,omitempty"`

	// +optional
	FrontPort string `json:"frontPort,omitempty"`

	// +optional
	Backends []BackEnd `json:"backends,omitempty"`

	// +optional
	BackPort string `json:"backPort,omitempty"`

	// +optional
	DomainName string `json:"domainName,omitempty"`

	// +optional
	Domains []Domain `json:"domains,omitempty"`
}

type Domain struct {
	// +optional
	DomainName string `json:"domainName,omitempty"`

	// +optional
	DomainId string `json:"domainId,omitempty"`

	// +optional
	BackPort string `json:"backPort,omitempty"`

	// +optional
	Backends BackEnd `json:"backends,omitempty"`
}

type BackEnd struct {
	// +optional
	VmIdentifier string `json:"vmIdentifier,omitempty"`

	// +optional
	IP string `json:"ip,omitempty"`
}

type VPCSpec struct {
	// ID is the vpc-id of the VPC this provider should use to create resources.
	// +optional
	ID string `json:"id,omitempty"`

	// +optional
	Name string `json:"name"`

	// +optional
	DcIdentifier string `json:"dcIdentifier"`

	// +optional
	NetworkRange string `json:"networkRange,omitempty"`

	// +optional
	NetworkSize string `json:"networkSize,omitempty"`

	// +optional
	AutoGenerate int `json:"autoGenerate,omitempty"`
}

var (
	// DefaultLBPort default LoadBalancer port.
	DefaultLBPort = 6443
	// DefaultLBAlgorithm default LoadBalancer algorithm.
	DefaultLBAlgorithm = "round_robin"
	// DefaultLBHealthCheckInterval default LoadBalancer health check interval.
	DefaultLBHealthCheckInterval = 1000
	// DefaultLBHealthCheckTimeout default LoadBalancer health check timeout.
	DefaultLBHealthCheckTimeout = 500
	// DefaultLBHealthCheckUnhealthyThreshold default LoadBalancer unhealthy threshold.
	DefaultLBHealthCheckUnhealthyThreshold = 2
	// DefaultLBHealthCheckHealthyThreshold default LoadBalancer healthy threshold.
	DefaultLBHealthCheckHealthyThreshold = 5
)

func (v *LoadBalancer) ApplyDefaults() {
	if v.Algorithm == "" {
		v.Algorithm = DefaultLBAlgorithm
	}

	if v.HealthCheck.HealthyThreshold == 0 {
		v.HealthCheck.HealthyThreshold = DefaultLBHealthCheckHealthyThreshold
	}

	if v.HealthCheck.HealthyPath == "" {
		v.HealthCheck.HealthyPath = "/"
	}

	if v.HealthCheck.UnhealthyThreshold == 0 {
		v.HealthCheck.UnhealthyThreshold = DefaultLBHealthCheckUnhealthyThreshold
	}

	if v.HealthCheck.Timeout == 0 {
		v.HealthCheck.Timeout = DefaultLBHealthCheckTimeout
	}

	if v.HealthCheck.Interval == 0 {
		v.HealthCheck.Interval = DefaultLBHealthCheckInterval
	}
}

func SafeName(name string) string {
	r := strings.NewReplacer(".", "-", "/", "-")
	return r.Replace(name)
}

// VpsieLoadBalancerHealthCheck define the Vpsie loadbalancers health check configurations.
type VpsieLoadBalancerHealthCheck struct {

	// The number of miilliseconds between between two consecutive health checks
	// If not specified, the default value is 1000.
	// +optional
	Interval int `json:"interval,omitempty"`

	// The number of milliseconds the Load Balancer instance will wait for a response until marking a health check as failed.
	// If not specified, the default value is 500.
	// +optional
	Timeout int `json:"timeout,omitempty"`

	// The number of times a health check must fail for a backend Droplet to be marked "unhealthy" and be removed from the pool.
	// The vaule must be between 2 and 10. If not specified, the default value is 2.
	// +optional
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=10
	UnhealthyThreshold int `json:"unhealthyThreshold,omitempty"`

	// The number of times a health check must pass for a backend Droplet to be marked "healthy" and be re-added to the pool.
	// The vaule must be between 2 and 10. If not specified, the default value is 5.
	// +optional
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=10
	HealthyThreshold int `json:"healthyThreshold,omitempty"`

	//  +optional
	HealthyPath string `json:"healthypath,omitempty"`
}
