package events

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/openfaas-incubator/connector-sdk/types"
	"github.com/pkg/errors"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/event"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	vtypes "github.com/vmware/govmomi/vim25/types"
)

// OutboundEvent is the JSON object sent to subscribed functions
// If the ManagedObjectReference for an event cannot be retrieved, it will be nil and thus not marshaled into the JSON OutboundEvent
// It's the receivers responsibility to check whether managedObjectReference key is present in the JSON message payload
type OutboundEvent struct {
	Topic    string `json:"topic,omitempty"`
	Category string `json:"category,omitempty"`

	UserName               string                         `json:"userName,omitempty"`
	CreatedTime            time.Time                      `json:"createdTime,omitempty"`
	ManagedObjectReference *vtypes.ManagedObjectReference `json:"managedObjectReference,omitempty"`
}

// NewVCenterClient returns a govmomi.Client to connect to vCenter
func NewVCenterClient(ctx context.Context, user string, pass string, vcenterURL string, insecure bool) (*govmomi.Client, error) {
	u, err := soap.ParseURL(vcenterURL)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing URL")
	}

	u.User = url.UserPassword(user, pass)
	return govmomi.NewClient(ctx, u, insecure)
}

// Stream is the main logic, blocking to receive and handle events from vCenter
func Stream(ctx context.Context, c *vim25.Client, controller *types.Controller) error {
	// create event manager to consume events from vCenter
	m := event.NewManager(c)

	// get events for all types (i.e. RootFolder)
	managedTypes := []vtypes.ManagedObjectReference{c.ServiceContent.RootFolder}
	eventsPerPage := int32(1)
	tail := true
	force := true

	recv := makeRecv(controller, m)
	err := m.Events(ctx, managedTypes, eventsPerPage, tail, force, recv)
	if err != nil {
		return errors.Wrap(err, "error connecting to event-stream")
	}
	return nil
}

// makeRecv returns a event handler function called by the event manager on each event
func makeRecv(controller *types.Controller, m *event.Manager) func(managedObjectReference vtypes.ManagedObjectReference, baseEvent []vtypes.BaseEvent) error {
	return func(managedObjectReference vtypes.ManagedObjectReference, baseEvent []vtypes.BaseEvent) error {
		log.Printf("Object %v", managedObjectReference)

		for i, event := range baseEvent {
			log.Printf("Event [%d] %v", i, event)

			topic, message, err := handleEvent(event, m)
			if err != nil {
				log.Printf("error handling event: %s", err.Error())
				continue
			}
			binaryMsg := []byte(message)
			log.Printf("Message on topic: %s", topic)
			controller.Invoke(topic, &binaryMsg)

		}
		return nil
	}
}

func handleEvent(event vtypes.BaseEvent, m *event.Manager) (string, string, error) {
	// Sanity check to avoid nil pointer exception
	if event == nil {
		return "", "", errors.New("event must not be nil")
	}

	// Get the type of the event, e.g. "VmPoweredOnEvent" which we'll use for subscribed topic matching
	eventType := reflect.TypeOf(event).Elem().Name()
	topic := convertToTopic(eventType)

	// Retrieve user name and category from the event
	user := event.GetEvent().UserName
	createdTime := event.GetEvent().CreatedTime
	category, err := m.EventCategory(context.Background(), event)
	if err != nil {
		return "", "", errors.Wrap(err, "error retrieving event category")
	}

	// Get the ManagedObjectReference by converting the event into a concrete event
	// If we don't find a MoRef in the event, *ref will be nil and not marshaled in the OutboundEvent making it easy for the subscribed function to validate the JSON payload
	ref := getMoref(event)

	message, err := json.Marshal(OutboundEvent{
		Topic:                  topic,
		Category:               category,
		UserName:               user,
		CreatedTime:            createdTime,
		ManagedObjectReference: ref,
	})
	if err != nil {
		return "", "", errors.Wrap(err, "error marshaling outboundevent")
	}

	log.Printf("message: %s", string(message))
	return topic, string(message), nil
}

// getMoref extracts the ManagedObjectReference, if any, by converting the BaseEvent to a concrete event
// Details on the ManagedObjectReference:
// https://code.vmware.com/docs/7371/vmware-vsphere-web-services-sdk-programming-guide-6-7-update-1#/doc/GUID-C9E81F17-2516-49EE-914F-EE9904258ED3.html
func getMoref(event vtypes.BaseEvent) *vtypes.ManagedObjectReference {
	// This pointer to the ManagedObjectReference will be used during the BaseEvent switch below
	// If we don't find a MoRef in the event, *ref will be nil and not marshaled in the OutboundEvent
	var ref *vtypes.ManagedObjectReference

	// Get the underlying concrete base event, e.g. VmEvent
	// vSphere Web Service API Reference 6.7
	// https://code.vmware.com/apis/358/vsphere#/doc/vim.event.Event.html
	switch baseEvent := event.(type) {
	// Alerts
	case vtypes.BaseAlarmEvent:
		e := baseEvent.GetAlarmEvent()
		ref = &e.Alarm.Alarm

	// Datastore
	case vtypes.BaseDatastoreEvent:
		e := baseEvent.GetDatastoreEvent()
		ref = &e.Datastore.Datastore

	// Host/ESX
	case vtypes.BaseHostEvent:
		e := baseEvent.GetHostEvent()
		ref = &e.Host.Host

	// Resource Pool
	case vtypes.BaseResourcePoolEvent:
		e := baseEvent.GetResourcePoolEvent()
		ref = &e.ResourcePool.ResourcePool

	// VM
	case vtypes.BaseVmEvent:
		e := baseEvent.GetVmEvent()
		ref = &e.Vm.Vm
	}

	return ref
}

// convertToTopic converts an event type to an OpenFaaS subscriber topic, e.g. "VmPoweredOnEvent" to "vm.powered.on"
func convertToTopic(eventType string) string {
	eventType = strings.Replace(eventType, "Event", "", -1)
	return camelCaseToLowerSeparated(eventType, ".")
}

// From https://github.com/vmware/dispatch/blob/master/pkg/utils/str_convert.go
// camelCaseToLowerSeparated converts a camel cased string to a multi-word string delimited by the specified separator
func camelCaseToLowerSeparated(src string, sep string) string {
	var words []string
	var word []rune
	var last rune
	for _, r := range src {
		if unicode.IsUpper(r) {
			if unicode.IsUpper(last) {
				// We have two uppercase letters in a row, it might be uppercase word like ID or SDK
				word = append(word, r)
			} else {
				// We have uppercase after lowercase, which always means start of a new word
				if len(word) > 0 {
					words = append(words, strings.ToLower(string(word)))
				}
				word = []rune{r}
			}
		} else {
			if unicode.IsUpper(last) && len(word) >= 2 {
				// We have a multi-uppercase word followed by an another word, e.g. "SDKToString",
				// but word variable contains "SDKT". We need to extract "T" as a first letter of a new word
				words = append(words, strings.ToLower(string(word[:len(word)-1])))
				word = []rune{last, r}
			} else {
				word = append(word, r)
			}
		}
		last = r
	}
	if len(word) > 0 {
		words = append(words, strings.ToLower(string(word)))
	}
	return strings.Join(words, sep)
}
