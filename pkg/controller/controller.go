package controller

import (
	"github.com/pkg/errors"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/util/iptables"
	"log"
	"sync"
)

type NetworkPolicyController interface {
	AddNamespace(ns *api.Namespace) error
	UpdateNamespace(oldObj, newObj *api.Namespace) error
	DeleteNamespace(ns *api.Namespace) error

	AddPod(obj *api.Pod) error
	UpdatePod(oldObj, newObj *api.Pod) error
	DeletePod(obj *api.Pod) error

	AddNetworkPolicy(obj *extensions.NetworkPolicy) error
	UpdateNetworkPolicy(oldObj, newObj *extensions.NetworkPolicy) error
	DeleteNetworkPolicy(obj *extensions.NetworkPolicy) error
}

type controller struct {
	sync.Mutex

	ipt         iptables.Interface
	nss         map[string]*ns // ns name -> ns struct
	nsSelectors selectorSet    // selector string -> nsSelector
}

func New(ipt iptables.Interface) NetworkPolicyController {
	return &controller{
		ipt:         ipt,
		nss:         make(map[string]*ns),
		nsSelectors: newSelectorSet()}
}

func (npc *controller) withNS(name string, f func(ns *ns) error) error {
	ns, found := npc.nss[name]
	if !found {
		newNs, err := newNS(name, npc.ipt, npc.nss, npc.nsSelectors)
		if err != nil {
			return err
		}
		npc.nss[name] = newNs
		ns = newNs
	}
	if err := f(ns); err != nil {
		return err
	}
	if ns.empty() {
		delete(npc.nss, name)
	}
	return nil
}

func (npc *controller) AddPod(obj *api.Pod) error {
	npc.Lock()
	defer npc.Unlock()

	return npc.withNS(obj.ObjectMeta.Namespace, func(ns *ns) error {
		return errors.Wrap(ns.addPod(obj), "add pod")
	})
}

func (npc *controller) UpdatePod(oldObj, newObj *api.Pod) error {
	npc.Lock()
	defer npc.Unlock()

	return npc.withNS(oldObj.ObjectMeta.Namespace, func(ns *ns) error {
		return errors.Wrap(ns.updatePod(oldObj, newObj), "update pod")
	})
}

func (npc *controller) DeletePod(obj *api.Pod) error {
	npc.Lock()
	defer npc.Unlock()

	return npc.withNS(obj.ObjectMeta.Namespace, func(ns *ns) error {
		return errors.Wrap(ns.deletePod(obj), "delete pod")
	})
}

func (npc *controller) AddNetworkPolicy(obj *extensions.NetworkPolicy) error {
	npc.Lock()
	defer npc.Unlock()

	return npc.withNS(obj.ObjectMeta.Namespace, func(ns *ns) error {
		return errors.Wrap(ns.addNetworkPolicy(obj), "add network policy")
	})
}

func (npc *controller) UpdateNetworkPolicy(oldObj, newObj *extensions.NetworkPolicy) error {
	npc.Lock()
	defer npc.Unlock()

	return npc.withNS(oldObj.ObjectMeta.Namespace, func(ns *ns) error {
		log.Println("Updating network policy from %v to %v", oldObj, newObj)
		return errors.Wrap(ns.updateNetworkPolicy(oldObj, newObj), "update network policy")
	})
}

func (npc *controller) DeleteNetworkPolicy(obj *extensions.NetworkPolicy) error {
	npc.Lock()
	defer npc.Unlock()

	return npc.withNS(obj.ObjectMeta.Namespace, func(ns *ns) error {
		return errors.Wrap(ns.deleteNetworkPolicy(obj), "delete network policy")
	})
}

func (npc *controller) AddNamespace(obj *api.Namespace) error {
	npc.Lock()
	defer npc.Unlock()

	return npc.withNS(obj.ObjectMeta.Name, func(ns *ns) error {
		return errors.Wrap(ns.addNamespace(obj), "add namespace")
	})
}

func (npc *controller) UpdateNamespace(oldObj, newObj *api.Namespace) error {
	npc.Lock()
	defer npc.Unlock()

	return npc.withNS(oldObj.ObjectMeta.Name, func(ns *ns) error {
		return errors.Wrap(ns.updateNamespace(oldObj, newObj), "update namespace")
	})
}

func (npc *controller) DeleteNamespace(obj *api.Namespace) error {
	npc.Lock()
	defer npc.Unlock()

	return npc.withNS(obj.ObjectMeta.Name, func(ns *ns) error {
		return errors.Wrap(ns.deleteNamespace(obj), "delete namespace")
	})
}
