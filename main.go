package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/openfaas-incubator/connector-sdk/types"
	"github.com/openfaas/faas-provider/auth"
	"github.com/pkg/errors"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/event"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	vtypes "github.com/vmware/govmomi/vim25/types"
)

func main() {
	var vcenterURL string
	var insecure bool

	flag.StringVar(&vcenterURL, "vcenter-url", "", "URL for vcenter user:password@domain.com:port")
	flag.Bool("insecure", true, "use an insecure connection")
	flag.Parse()

	if len(vcenterURL) == 0 {
		panic("vcenterURL not provided")
	}

	vcenterClient, vcenterErr := newVCenterClient(context.Background(), vcenterURL, insecure)

	if vcenterErr != nil {
		panic(vcenterErr)
	}

	var credentials *auth.BasicAuthCredentials

	if val, ok := os.LookupEnv("basic_auth"); ok && len(val) > 0 {
		if val == "true" || val == "1" {

			reader := auth.ReadBasicAuthFromDisk{}

			if val, ok := os.LookupEnv("secret_mount_path"); ok && len(val) > 0 {
				reader.SecretMountPath = os.Getenv("secret_mount_path")
			}

			res, err := reader.Read()
			if err != nil {
				panic(err)
			}
			credentials = res
		}
	}

	config := types.ControllerConfig{
		GatewayURL:      os.Getenv("OPENFAAS_URL"),
		PrintResponse:   false,
		RebuildInterval: time.Second * 10,
		UpstreamTimeout: time.Second * 15,
	}

	controller := types.NewController(credentials, &config)

	controller.BeginMapBuilder()

	err := bindEvents(vcenterClient.Client, controller)

	if err != nil {
		panic(err)
	}
}

func bindEvents(c *vim25.Client, controller *types.Controller) error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := event.NewManager(c)
	managedTypes := []vtypes.ManagedObjectReference{c.ServiceContent.RootFolder}
	eventsPerPage := int32(1)
	tail := true
	force := true

	recv := makeRecv(controller, m)
	go func() {
		err := m.Events(ctx, managedTypes, eventsPerPage, tail, force, recv)
		if err != nil {
			log.Printf("error connecting to event-stream: %v", err.Error())
			cancel()
		}
	}()
	<-ctx.Done()
	// done := make(chan bool)

	// <-done

	// controller.Invoke()

	return nil
}

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
	eventType := reflect.TypeOf(event).Elem().Name()

	gotEvent := event.GetEvent()
	if gotEvent.Vm != nil {
		log.Printf("VM: %s", gotEvent.Vm.Vm.Reference().String())
	}
	log.Printf("UserName: %s", gotEvent.UserName)

	category, err := m.EventCategory(context.Background(), event)
	if err != nil {
		return "", "", errors.Wrap(err, "error retrieving event category")
	}

	log.Printf("category: %v, event type: %v", category, eventType)

	topic := convertToTopic(eventType)

	message, _ := json.Marshal(OutboundEvent{
		Topic:       topic,
		Category:    category,
		CreatedTime: gotEvent.CreatedTime,
		UserName:    gotEvent.UserName,
	})

	return topic, string(message), nil
}

type OutboundEvent struct {
	Topic    string `json:"topic,omitempty"`
	Category string `json:"category,omitempty"`

	UserName    string    `json:"userName,omitempty"`
	CreatedTime time.Time `json:"createdTime,omitempty"`
}

func newVCenterClient(ctx context.Context, vcenterURL string, insecure bool) (*govmomi.Client, error) {
	url, err := soap.ParseURL(vcenterURL)
	if err != nil {
		return nil, err
	}

	return govmomi.NewClient(ctx, url, insecure)
}

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
