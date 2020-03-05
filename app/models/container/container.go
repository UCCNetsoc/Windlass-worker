package container

type Containers []Container

type Container struct {
	// Name of the container
	Name string `json:"name"`

	// The name of the image eg `ubuntu:18.04`
	Image string `json:"image"`

	// Command, if any, to start the container with
	Command string `json:"command,omitempty"`

	// The mapping between ports within the container and ports on the host
	// Only mapped ports will be accessible outside the container host
	Ports []PortMapping `json:"ports"`

	// Specifies the file/directory mappings from the host into the container
	Mounts []MountMapping `json:"mounts"`

	// User specified labels for the container
	Labels map[string]string `json:"labels"`

	// Environment variables to set for the container
	Env map[string]string `json:"env"`

	// Is the container started on creation
	Started bool `json:"started"`
}

type PortMapping struct {
	// Port within the container to map to a host port
	ContainerPort uint16 `json:"internalPort"`

	// The host port to map to the container port
	HostPort uint16 `json:"hostPort"`
}

type MountMapping struct {
	// Host mount point
	Source string `json:"source"`

	// Container mount point
	Destination string `json:"destination"`

	// If false, is equal to `/host:/container:ro`
	RW bool `json:"rw"`
}
