package events

import "testing"
import vtypes "github.com/vmware/govmomi/vim25/types"

var (
	vmEvent = &vtypes.VmEvent{
		Event: vtypes.Event{
			Vm: &vtypes.VmEventArgument{
				Vm: vtypes.ManagedObjectReference{
					Type:  "VirtualMachine",
					Value: "vm-1234",
				},
			},
		},
	}

	resourcePoolEvent = &vtypes.ResourcePoolEvent{
		ResourcePool: vtypes.ResourcePoolEventArgument{
			ResourcePool: vtypes.ManagedObjectReference{
				Type:  "ResourcePool",
				Value: "resgroup-1234",
			},
		},
	}

	unsupportedEvent = &vtypes.LicenseEvent{
		Event: vtypes.Event{},
	}
)

var tests = []struct {
	name    string
	event   vtypes.BaseEvent
	moref   *vtypes.ManagedObjectReference
	motype  string
	movalue string
}{
	{"valid VM Event", vmEvent, &vmEvent.Vm.Vm, "VirtualMachine", "vm-1234"},
	{"valid ResourcePool Event", resourcePoolEvent, &resourcePoolEvent.ResourcePool.ResourcePool, "ResourcePool", "resgroup-1234"},
	// assert that ManagedObjectReference will be nil for events we don't support (yet), so it won't be marshaled in the outbound JSON
	{"unsupported Event", unsupportedEvent, nil, "", ""},
}

func TestGetMoref(t *testing.T) {
	for _, test := range tests {
		t.Logf("running test: %q", test.name)
		ref := getMoref(test.event)
		if ref != test.moref {
			t.Errorf("Received incorrect MoRef, got: %v, want: %v", ref, test.moref)
		}

		if ref != nil {
			if ref.Value != test.movalue {
				t.Errorf("Received incorrect MoRef value, got: %v, want: %v", ref.Value, test.movalue)
			}
		}

	}
}
