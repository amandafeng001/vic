// Copyright 2016 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package network

import (
	"fmt"
	"net"

	"github.com/docker/docker/pkg/stringid"
	"github.com/vmware/vic/pkg/ip"
)

type alias struct {
	Name      string
	Container string

	ep *Endpoint
}

var badAlias = alias{}

type Endpoint struct {
	container *Container
	scope     *Scope
	ip        net.IP
	static    bool
	ports     map[Port]interface{} // exposed ports
	aliases   map[string][]alias
	id        string
}

// scopeName returns the "fully qualified" name of an alias. Aliases are scoped
// by the container and network scope they are in.
func (a alias) scopedName() string {
	// an alias for the container itself is network scoped
	for _, al := range a.ep.getAliases("") {
		if a.Name == al.Name {
			return ScopedAliasName(a.ep.Scope().Name(), "", a.Name)
		}
	}

	return ScopedAliasName(a.ep.Scope().Name(), a.ep.Container().Name(), a.Name)
}

// ScopedAliasName returns the fully qualified name of an alias, scoped to
// the scope and optionally a container
func ScopedAliasName(scope string, container string, alias string) string {
	if container != "" {
		return fmt.Sprintf("%s:%s:%s", scope, container, alias)
	}

	return fmt.Sprintf("%s:%s", scope, alias)
}

func newEndpoint(container *Container, scope *Scope, eip *net.IP, pciSlot *int32) *Endpoint {
	e := &Endpoint{
		container: container,
		scope:     scope,
		ip:        net.IPv4(0, 0, 0, 0),
		static:    false,
		ports:     make(map[Port]interface{}),
		aliases:   make(map[string][]alias),
		id:        stringid.GenerateRandomID(),
	}

	if eip != nil && !ip.IsUnspecifiedIP(*eip) {
		e.ip = *eip
		e.static = true
	}

	return e
}

func removeEndpointHelper(ep *Endpoint, eps []*Endpoint) []*Endpoint {
	for i, e := range eps {
		if ep != e {
			continue
		}

		return append(eps[:i], eps[i+1:]...)
	}

	return eps
}

func (e *Endpoint) addPort(p Port) error {
	if _, ok := e.ports[p]; ok {
		return fmt.Errorf("port %s already exposed", p)
	}

	e.ports[p] = nil
	return nil
}

func (e *Endpoint) IP() net.IP {
	return e.ip
}

func (e *Endpoint) Scope() *Scope {
	return e.scope
}

func (e *Endpoint) Subnet() *net.IPNet {
	return e.Scope().Subnet()
}

func (e *Endpoint) Container() *Container {
	return e.container
}

func (e *Endpoint) ID() string {
	return e.id
}

func (e *Endpoint) Name() string {
	return e.container.Name()
}

func (e *Endpoint) Gateway() net.IP {
	return e.Scope().Gateway()
}

func (e *Endpoint) Ports() []Port {
	ports := make([]Port, len(e.ports))
	i := 0
	for p := range e.ports {
		ports[i] = p
		i++
	}

	return ports
}

func (e *Endpoint) addAlias(con, a string) (alias, bool) {
	if a == "" {
		return badAlias, false
	}

	if con == "" {
		con = e.container.Name()
	}

	aliases := e.aliases[con]
	for _, as := range aliases {
		if as.Name == a {
			// already present
			return as, true
		}
	}

	na := alias{
		Name:      a,
		Container: con,
		ep:        e,
	}
	e.aliases[con] = append(aliases, na)
	return na, false
}

func (e *Endpoint) getAliases(con string) []alias {
	if con == "" {
		con = e.container.Name()
	}

	return e.aliases[con]
}

func (e *Endpoint) copy() *Endpoint {
	other := &Endpoint{}
	*other = *e
	other.ports = make(map[Port]interface{})
	for p := range e.ports {
		other.ports[p] = nil
	}

	return other
}
