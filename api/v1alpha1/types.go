package v1alpha1

type VpsieResourceStatus string

var (
	// VpsieResourceStatusNew is the string representing a Vpsie resource just created and in a provisioning state.
	VpsieResourceStatusNew = VpsieResourceStatus("new")
	// VpsieResourceStatusRunning is the string representing a Vpsie resource already provisioned and in a active state.
	VpsieResourceStatusRunning = VpsieResourceStatus("active")
	// VpsieResourceStatusErrored is the string representing a Vpsie resource in a errored state.
	VpsieResourceStatusErrored = VpsieResourceStatus("errored")
	// VpsieResourceStatusOff is the string representing a Vpsie resource in off state.
	VpsieResourceStatusOff = VpsieResourceStatus("off")
)

// DOResourceReference is a reference to a Vpsie resource.
type VpsieResourceReference struct {
	// ID of Vpsie resource
	// +optional
	ID string `json:"id,omitempty"`
	// Status of Vpsie resource
	// +optional
	Status VpsieResourceStatus `json:"status,omitempty"`
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
	RedirectHTTP string `json:"redirectHTTP,omitempty"`

	// +optional
	Rules []Rule `json:"rules,omitempty"`

	// +optional
	Domains []Domain `json:"domains,omitempty"`
}

type Rule struct {
	// +optional
	Scheme string `json:"scheme,omitempty"`

	// +optional
	FrontPort string `json:"frontPort,omitempty"`
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
