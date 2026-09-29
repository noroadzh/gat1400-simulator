package scenario

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

const sampleYAML = `
id: demo-person-flow
name: Person Flow Demo
description: device → viewlib → subscriber
tags: [demo, person]
nodes:
  - ref: dev
    role: device
    listen: ":14001"
    upstream: ":14002"
    capabilities: [collect, register]
  - ref: lib
    role: platform-small
    listen: ":14002"
    capabilities: [store]
subscriptions:
  - subscriber: lib
    publisher: dev
    topic: Person
    interval: 30s
    receiveAddr: http://127.0.0.1:14003/cb
resources:
  - node: dev
    kind: Person
    rate: 2
    burst: 5
    seed: 42
exceptions:
  registerDropRate: 0.0
  keepaliveJitter: 50ms
  subscribeRejectRate: 0.0
  latencySpike: 0s
schedule:
  autoStart: true
  duration: 0s
`

func TestScenario_UnmarshalYAML_Roundtrip(t *testing.T) {
	var s Scenario
	if err := yaml.Unmarshal([]byte(sampleYAML), &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if s.ID != "demo-person-flow" {
		t.Fatalf("ID = %q", s.ID)
	}
	if s.Name != "Person Flow Demo" {
		t.Fatalf("Name = %q", s.Name)
	}
	if len(s.Tags) != 2 || s.Tags[0] != "demo" || s.Tags[1] != "person" {
		t.Fatalf("Tags = %v", s.Tags)
	}
	if len(s.Nodes) != 2 {
		t.Fatalf("Nodes = %d, want 2", len(s.Nodes))
	}
	dev := s.Nodes[0]
	if dev.Ref != "dev" || dev.Role != node.RoleDevice || dev.Listen != ":14001" {
		t.Fatalf("dev spec = %+v", dev)
	}
	if len(dev.Capabilities) != 2 {
		t.Fatalf("dev capabilities = %v", dev.Capabilities)
	}
	if len(s.Subscriptions) != 1 {
		t.Fatalf("Subscriptions = %d, want 1", len(s.Subscriptions))
	}
	sub := s.Subscriptions[0]
	if sub.SubscriberRef != "lib" || sub.PublisherRef != "dev" || sub.Topic != resource.KindPerson {
		t.Fatalf("sub = %+v", sub)
	}
	if sub.Interval != 30*time.Second {
		t.Fatalf("sub.Interval = %s, want 30s", sub.Interval)
	}
	if len(s.Resources) != 1 || s.Resources[0].Kind != resource.KindPerson || s.Resources[0].Rate != 2 {
		t.Fatalf("resources = %+v", s.Resources)
	}
	if !s.Schedule.AutoStart || s.Schedule.Duration != 0 {
		t.Fatalf("schedule = %+v", s.Schedule)
	}
}

func TestScenario_ZeroValue_Defaults(t *testing.T) {
	var s Scenario
	if s.ID != "" || len(s.Nodes) != 0 {
		t.Fatalf("zero value should be empty, got %+v", s)
	}
}

func TestScenario_NodeSpec_DefaultsAreZeroValues(t *testing.T) {
	var n NodeSpec
	if n.Role != node.Role("") {
		t.Fatalf("Role zero value = %q", n.Role)
	}
	if n.Listen != "" || n.Upstream != "" {
		t.Fatalf("listen/upstream should default to empty")
	}
}

func TestScenario_ResourceSpec_DefaultsToZeroRate(t *testing.T) {
	var r ResourceSpec
	if r.Rate != 0 || r.Burst != 0 {
		t.Fatalf("rate/burst should default to 0, got %+v", r)
	}
}

func TestScenario_ExceptionSpec_DisabledByZeroValues(t *testing.T) {
	var e ExceptionSpec
	if e.RegisterDropRate != 0 || e.KeepaliveJitter != 0 || e.SubscribeRejectRate != 0 || e.LatencySpike != 0 {
		t.Fatalf("exception spec should default to all-zero, got %+v", e)
	}
}