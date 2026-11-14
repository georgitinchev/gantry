// Package api implements gantry-cloud: a deliberately small IaaS with the
// behaviour real cloud APIs have and providers must handle: asynchronous
// provisioning, quotas, IP exhaustion, conflicts and transient failures.
package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/netip"
	"sort"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("not_found")
	ErrConflict  = errors.New("conflict")
	ErrInUse     = errors.New("in_use")
	ErrQuota     = errors.New("quota_exceeded")
	ErrExhausted = errors.New("ip_exhausted")
	ErrOverlap   = errors.New("cidr_overlap")
	ErrInvalid   = errors.New("invalid")
	ErrBusy      = errors.New("not_running")
)

// Sizes a VM can have, smallest first.
var Sizes = []string{"small", "medium", "large"}

type Network struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CIDR      string    `json:"cidr"`
	CreatedAt time.Time `json:"createdAt"`
}

type Instance struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	NetworkID string    `json:"networkId"`
	Size      string    `json:"size"`
	IP        string    `json:"ip"`
	State     string    `json:"state"` // provisioning | running | resizing
	CreatedAt time.Time `json:"createdAt"`

	readyAt time.Time
}

// Config tunes the simulation.
type Config struct {
	Pool           netip.Prefix  // networks without an explicit CIDR get a /24 from here
	MaxInstances   int           // account-wide instance quota
	ProvisionDelay time.Duration // how long create/resize stays in a transitional state
	Now            func() time.Time
}

type Cloud struct {
	cfg       Config
	mu        sync.Mutex
	networks  map[string]*Network
	instances map[string]*Instance
}

func New(cfg Config) *Cloud {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Cloud{cfg: cfg, networks: map[string]*Network{}, instances: map[string]*Instance{}}
}

func newID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// ---- networks ----

func (c *Cloud) CreateNetwork(name, cidr string) (*Network, error) {
	if name == "" || len(name) > 63 {
		return nil, ErrInvalid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, n := range c.networks {
		if n.Name == name {
			return n, ErrConflict
		}
	}
	var p netip.Prefix
	if cidr == "" {
		var ok bool
		if p, ok = c.nextFreeSubnet(); !ok {
			return nil, ErrExhausted
		}
	} else {
		var err error
		if p, err = netip.ParsePrefix(cidr); err != nil || !p.Addr().Is4() || p.Bits() < 16 || p.Bits() > 28 || p.Masked() != p {
			return nil, ErrInvalid
		}
		if c.overlaps(p) {
			return nil, ErrOverlap
		}
	}
	n := &Network{ID: newID("net"), Name: name, CIDR: p.String(), CreatedAt: c.cfg.Now().UTC()}
	c.networks[n.ID] = n
	return n, nil
}

func (c *Cloud) overlaps(p netip.Prefix) bool {
	for _, n := range c.networks {
		if netip.MustParsePrefix(n.CIDR).Overlaps(p) {
			return true
		}
	}
	return false
}

// nextFreeSubnet walks the pool in /24 steps and returns the first unused block.
func (c *Cloud) nextFreeSubnet() (netip.Prefix, bool) {
	addr := c.cfg.Pool.Masked().Addr()
	for c.cfg.Pool.Contains(addr) {
		p := netip.PrefixFrom(addr, 24)
		if !c.overlaps(p) {
			return p, true
		}
		b := addr.As4()
		if b[2] == 255 {
			break
		}
		b[2]++
		addr = netip.AddrFrom4(b)
	}
	return netip.Prefix{}, false
}

func (c *Cloud) GetNetwork(id string) (*Network, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n, ok := c.networks[id]; ok {
		return n, nil
	}
	return nil, ErrNotFound
}

func (c *Cloud) ListNetworks() []*Network {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*Network, 0, len(c.networks))
	for _, n := range c.networks {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (c *Cloud) DeleteNetwork(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.networks[id]; !ok {
		return ErrNotFound
	}
	for _, i := range c.instances {
		if i.NetworkID == id {
			return ErrInUse
		}
	}
	delete(c.networks, id)
	return nil
}

// ---- instances ----

func validSize(s string) bool {
	for _, v := range Sizes {
		if v == s {
			return true
		}
	}
	return false
}

func (c *Cloud) CreateInstance(name, networkID, size string) (*Instance, error) {
	if name == "" || len(name) > 63 || !validSize(size) {
		return nil, ErrInvalid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, i := range c.instances {
		if i.Name == name {
			return c.refresh(i), ErrConflict
		}
	}
	n, ok := c.networks[networkID]
	if !ok {
		return nil, ErrNotFound
	}
	if len(c.instances) >= c.cfg.MaxInstances {
		return nil, ErrQuota
	}
	ip, ok := c.nextFreeIP(n)
	if !ok {
		return nil, ErrExhausted
	}
	now := c.cfg.Now().UTC()
	i := &Instance{ID: newID("vm"), Name: name, NetworkID: n.ID, Size: size, IP: ip.String(), State: "provisioning", CreatedAt: now, readyAt: now.Add(c.cfg.ProvisionDelay)}
	c.instances[i.ID] = i
	return c.refresh(i), nil
}

// nextFreeIP hands out host addresses from .10 up, leaving the bottom of the
// range for gateways the way most clouds do.
func (c *Cloud) nextFreeIP(n *Network) (netip.Addr, bool) {
	p := netip.MustParsePrefix(n.CIDR)
	used := map[netip.Addr]bool{}
	for _, i := range c.instances {
		if i.NetworkID == n.ID {
			used[netip.MustParseAddr(i.IP)] = true
		}
	}
	a := p.Addr()
	for k := 0; k < 10; k++ {
		a = a.Next()
	}
	for ; p.Contains(a); a = a.Next() {
		if !p.Contains(a.Next()) { // broadcast
			break
		}
		if !used[a] {
			return a, true
		}
	}
	return netip.Addr{}, false
}

// refresh advances time-based state; callers hold the lock.
func (c *Cloud) refresh(i *Instance) *Instance {
	if i.State != "running" && !c.cfg.Now().Before(i.readyAt) {
		i.State = "running"
	}
	cp := *i
	return &cp
}

func (c *Cloud) GetInstance(id string) (*Instance, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if i, ok := c.instances[id]; ok {
		return c.refresh(i), nil
	}
	return nil, ErrNotFound
}

func (c *Cloud) ListInstances() []*Instance {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*Instance, 0, len(c.instances))
	for _, i := range c.instances {
		out = append(out, c.refresh(i))
	}
	sort.Slice(out, func(a, b int) bool { return out[a].CreatedAt.Before(out[b].CreatedAt) })
	return out
}

// ResizeInstance changes the size of a running instance; it passes through
// "resizing" like a real stop/resize/start cycle.
func (c *Cloud) ResizeInstance(id, size string) (*Instance, error) {
	if !validSize(size) {
		return nil, ErrInvalid
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	i, ok := c.instances[id]
	if !ok {
		return nil, ErrNotFound
	}
	c.refresh(i)
	if i.Size == size {
		return c.refresh(i), nil
	}
	if i.State != "running" {
		return nil, ErrBusy
	}
	i.Size, i.State, i.readyAt = size, "resizing", c.cfg.Now().Add(c.cfg.ProvisionDelay)
	return c.refresh(i), nil
}

func (c *Cloud) DeleteInstance(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.instances[id]; !ok {
		return ErrNotFound
	}
	delete(c.instances, id)
	return nil
}

// Counts reports instances by state, for metrics.
func (c *Cloud) Counts() (networks int, byState map[string]int) {
	byState = map[string]int{"provisioning": 0, "running": 0, "resizing": 0}
	for _, i := range c.ListInstances() {
		byState[i.State]++
	}
	return len(c.ListNetworks()), byState
}
