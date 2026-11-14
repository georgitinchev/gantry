package api

import (
	"errors"
	"net/netip"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time      { return c.t }
func (c *clock) add(d time.Duration) { c.t = c.t.Add(d) }

func newCloud(max int) (*Cloud, *clock) {
	clk := &clock{t: time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)}
	return New(Config{Pool: netip.MustParsePrefix("10.40.0.0/16"), MaxInstances: max, ProvisionDelay: 5 * time.Second, Now: clk.now}), clk
}

func TestNetworksGetDistinctSubnetsFromThePool(t *testing.T) {
	c, _ := newCloud(10)
	a, _ := c.CreateNetwork("a", "")
	b, _ := c.CreateNetwork("b", "")
	if a.CIDR != "10.40.0.0/24" || b.CIDR != "10.40.1.0/24" {
		t.Fatalf("got %s and %s", a.CIDR, b.CIDR)
	}
	if err := c.DeleteNetwork(a.ID); err != nil {
		t.Fatal(err)
	}
	again, _ := c.CreateNetwork("c", "")
	if again.CIDR != "10.40.0.0/24" {
		t.Fatalf("freed subnet should be reused, got %s", again.CIDR)
	}
}

func TestNetworkValidation(t *testing.T) {
	c, _ := newCloud(10)
	if _, err := c.CreateNetwork("x", "10.40.0.0/24"); err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		name, cidr string
		want       error
	}{
		"overlap":     {"y", "10.40.0.128/25", ErrOverlap},
		"not network": {"y", "10.41.0.5/24", ErrInvalid},
		"too large":   {"y", "10.0.0.0/8", ErrInvalid},
		"ipv6":        {"y", "fd00::/64", ErrInvalid},
		"empty name":  {"", "", ErrInvalid},
		"duplicate":   {"x", "", ErrConflict},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := c.CreateNetwork(tc.name, tc.cidr); !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestDuplicateCreateReturnsExistingForAdoption(t *testing.T) {
	c, _ := newCloud(10)
	n, _ := c.CreateNetwork("net", "")
	i, _ := c.CreateInstance("vm", n.ID, "small")
	again, err := c.CreateInstance("vm", n.ID, "small")
	if !errors.Is(err, ErrConflict) || again.ID != i.ID {
		t.Fatalf("want conflict with existing %s, got %v %+v", i.ID, err, again)
	}
}

func TestInstancesProvisionAsynchronously(t *testing.T) {
	c, clk := newCloud(10)
	n, _ := c.CreateNetwork("net", "")
	i, _ := c.CreateInstance("vm", n.ID, "small")
	if i.State != "provisioning" || i.IP != "10.40.0.10" {
		t.Fatalf("got %+v", i)
	}
	clk.add(5 * time.Second)
	if got, _ := c.GetInstance(i.ID); got.State != "running" {
		t.Fatalf("want running, got %s", got.State)
	}
}

func TestResizeGoesThroughTransitionalState(t *testing.T) {
	c, clk := newCloud(10)
	n, _ := c.CreateNetwork("net", "")
	i, _ := c.CreateInstance("vm", n.ID, "small")
	if _, err := c.ResizeInstance(i.ID, "medium"); !errors.Is(err, ErrBusy) {
		t.Fatalf("resize while provisioning should fail, got %v", err)
	}
	clk.add(5 * time.Second)
	r, err := c.ResizeInstance(i.ID, "medium")
	if err != nil || r.State != "resizing" || r.Size != "medium" {
		t.Fatalf("got %+v %v", r, err)
	}
	clk.add(5 * time.Second)
	if got, _ := c.GetInstance(i.ID); got.State != "running" {
		t.Fatalf("want running after resize, got %s", got.State)
	}
}

func TestQuotaAndIPExhaustion(t *testing.T) {
	c, _ := newCloud(2)
	n, _ := c.CreateNetwork("net", "")
	c.CreateInstance("a", n.ID, "small")
	c.CreateInstance("b", n.ID, "small")
	if _, err := c.CreateInstance("c", n.ID, "small"); !errors.Is(err, ErrQuota) {
		t.Fatalf("want quota error, got %v", err)
	}

	c2, _ := newCloud(100)
	tiny, _ := c2.CreateNetwork("tiny", "10.40.9.0/28") // .10 .. .14 usable
	for k := 0; k < 5; k++ {
		if _, err := c2.CreateInstance(string(rune('a'+k)), tiny.ID, "small"); err != nil {
			t.Fatalf("instance %d: %v", k, err)
		}
	}
	if _, err := c2.CreateInstance("z", tiny.ID, "small"); !errors.Is(err, ErrExhausted) {
		t.Fatalf("want exhaustion, got %v", err)
	}
}

func TestNetworkInUseCannotBeDeleted(t *testing.T) {
	c, _ := newCloud(10)
	n, _ := c.CreateNetwork("net", "")
	i, _ := c.CreateInstance("vm", n.ID, "small")
	if err := c.DeleteNetwork(n.ID); !errors.Is(err, ErrInUse) {
		t.Fatalf("want in_use, got %v", err)
	}
	c.DeleteInstance(i.ID)
	if err := c.DeleteNetwork(n.ID); err != nil {
		t.Fatal(err)
	}
}
