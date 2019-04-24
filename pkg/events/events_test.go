package events

import (
	"reflect"
	"testing"

	vtypes "github.com/vmware/govmomi/vim25/types"
)

var (
	vmEvent = &vtypes.VmEvent{
		Event: vtypes.Event{
			Vm: &vtypes.VmEventArgument{
				EntityEventArgument: vtypes.EntityEventArgument{
					Name: "Windows10-1234",
				},
				Vm: vtypes.ManagedObjectReference{
					Type:  "VirtualMachine",
					Value: "vm-1234",
				},
			},
		},
	}

	resourcePoolEvent = &vtypes.ResourcePoolEvent{
		ResourcePool: vtypes.ResourcePoolEventArgument{
			EntityEventArgument: vtypes.EntityEventArgument{
				Name: "Management-RP-1234",
			},
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

func TestGetObjectNameAndMoRef(t *testing.T) {

	var testCases = []struct {
		name        string
		event       vtypes.BaseEvent
		wantMoref   *vtypes.ManagedObjectReference
		wantObjName string
	}{
		{"valid VM Event", vmEvent, &vmEvent.Vm.Vm, "Windows10-1234"},
		{"valid ResourcePool Event", resourcePoolEvent, &resourcePoolEvent.ResourcePool.ResourcePool, "Management-RP-1234"},
		// assert that ManagedObjectReference and ObjectName will be nil for events we don't support (yet), so it won't be marshaled in the outbound JSON
		{"unsupported Event", unsupportedEvent, nil, ""},
	}

	for _, test := range testCases {
		name, ref := getObjectNameAndMoref(test.event)

		// test MoRef
		if eq := reflect.DeepEqual(test.wantMoref, ref); !eq {
			t.Errorf("%s: wanted: %v, got: %v", test.name, test.wantMoref, ref)
		}

		// test ObjectName
		if eq := reflect.DeepEqual(test.wantObjName, name); !eq {
			t.Errorf("%s: wanted: %v, got: %v", test.name, test.wantObjName, name)
		}
	}
}
