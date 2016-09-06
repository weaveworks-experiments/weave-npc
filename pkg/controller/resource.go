package controller

import (
	"k8s.io/kubernetes/pkg/types"
)

type ResourceKey string

type ResourceSpec interface {
	Key() ResourceKey
	Metadata() map[string]string
}

type Resource interface {
	Spec() ResourceSpec
}

type ResourceOps interface {
	Create(spec ResourceSpec) (Resource, error)
	Destroy(resource Resource) error
}

type ResourceManager interface {
	UpdateUsage(user types.UID, current, desired map[ResourceKey]ResourceSpec) error
}

type resourceManager struct {
	ops       ResourceOps
	users     map[ResourceKey]map[types.UID]struct{}
	resources map[ResourceKey]Resource
}

func NewResourceManager(ops ResourceOps) ResourceManager {
	return &resourceManager{
		ops:       ops,
		users:     make(map[ResourceKey]map[types.UID]struct{}),
		resources: make(map[ResourceKey]Resource)}
}

func (rm *resourceManager) UpdateUsage(user types.UID, current, desired map[ResourceKey]ResourceSpec) error {
	// Unreference (destroying if necessary) resources no longer needed by user
	for key, _ := range current {
		if _, found := desired[key]; !found {
			delete(rm.users[key], user)
			if len(rm.users[key]) == 0 {
				if err := rm.ops.Destroy(rm.resources[key]); err != nil {
					return err
				}
				delete(rm.resources, key)
				delete(rm.users, key)
			}
		}
	}

	// Reference (creating if necessary) resources now needed by user
	for key, spec := range desired {
		if _, found := current[key]; !found {
			if _, found := rm.resources[key]; !found {
				resource, err := rm.ops.Create(spec)
				if err != nil {
					return err
				}
				rm.resources[key] = resource
				rm.users[key] = make(map[types.UID]struct{})
			}
			rm.users[key][user] = struct{}{}
		}
	}

	return nil
}
