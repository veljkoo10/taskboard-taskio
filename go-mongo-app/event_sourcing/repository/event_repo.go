package repository

import (
	"context"
	"encoding/json"
	"errors"
	model "event_sourcing/models"
	"fmt"
	"log"
	"time"

	"github.com/EventStore/EventStore-Client-Go/esdb"
	"github.com/gofrs/uuid"
)

var (
	ErrEmptyStream    = errors.New("no events in the stream")
	ErrStreamNotFound = errors.New("stream not found")
)

type ESDBClient struct {
	client *esdb.Client
	group  string
	sub    *esdb.PersistentSubscription // Initialized in the subscribe method
}

// NewESDBClient initializes a new ESDBClient.
func NewESDBClient(client *esdb.Client, group string) (*ESDBClient, error) {
	opts := esdb.PersistentAllSubscriptionOptions{
		From: esdb.Start{},
	}
	// Attempt to create the subscription
	err := client.CreatePersistentSubscriptionAll(context.Background(), group, opts)
	if err != nil {
		// persistent subscription group already exists
		log.Println(err)
	}

	esdbClient := &ESDBClient{
		client: client,
		group:  group,
	}

	// Ensure the subscription is set up
	if err := esdbClient.subscribe(); err != nil {
		log.Println("Subscription error:", err)
		return nil, err
	}
	return esdbClient, nil
}

func (e *ESDBClient) StoreEvent(stream string, event model.Event) error {
	id, err := uuid.NewV4()
	if err != nil {
		return err
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Log the stream name and event details
	log.Printf("Storing event to stream: %s, event: %+v\n", stream, event)

	esEvent := esdb.EventData{
		EventID:     id,
		EventType:   string(event.Type),
		Data:        eventData,
		ContentType: esdb.JsonContentType,
	}
	opts := esdb.AppendToStreamOptions{}
	_, err = e.client.AppendToStream(context.Background(), stream, opts, esEvent)
	return err
}

// ProcessEvents processes events using a provided function.
func (e *ESDBClient) ProcessEvents(processFn func(event model.Event) error) {
	for {
		receivedEvent := e.sub.Recv()

		if receivedEvent.EventAppeared != nil {
			streamEvent := receivedEvent.EventAppeared.Event
			var event model.Event
			if err := json.Unmarshal(streamEvent.Data, &event); err != nil {
				log.Println("Failed to deserialize event:", err)
				e.sub.Nack(err.Error(), esdb.Nack_Park, receivedEvent.EventAppeared)
				continue
			}
			err := processFn(event)
			if err != nil {
				log.Println("Processing error:", err)
				e.sub.Nack(err.Error(), esdb.Nack_Retry, receivedEvent.EventAppeared)
			} else {
				e.sub.Ack(receivedEvent.EventAppeared)
			}
		}

		if receivedEvent.SubscriptionDropped != nil {
			log.Println("Subscription dropped:", receivedEvent.SubscriptionDropped.Error)
			for err := e.subscribe(); err != nil; {
				log.Println("Reattempting subscription in 5 seconds...")
				time.Sleep(5 * time.Second)
			}
		}
	}
}

// subscribe connects to the persistent subscription and assigns it to the sub field.
func (e *ESDBClient) subscribe() error {
	opts := esdb.ConnectToPersistentSubscriptionOptions{}
	sub, err := e.client.ConnectToPersistentSubscriptionToAll(context.Background(), e.group, opts)
	if err != nil {
		return err
	}
	e.sub = sub
	return nil
}

// GetEventsByProjectID retrieves all events associated with the given project ID from EventStoreDB
func (repo *ESDBClient) GetEventsByProjectID(projectID string) ([]model.Event, error) {
	// Stream name could be based on the project ID. For example, "project-{projectID}"
	streamName := fmt.Sprintf(projectID)

	// Create an empty slice to store the events
	var events []model.Event

	// Set up the options for reading events
	readOpts := esdb.ReadStreamOptions{
		From: esdb.Start{}, // Start from the beginning of the stream
	}

	// Specify the number of events you want to read (or adjust based on your needs)
	count := uint64(100) // Adjust this number based on your needs (or handle pagination)

	// Create a context (optional timeout can be added)
	ctx := context.Background()

	// Open the stream for reading with the specified options
	stream, err := repo.client.ReadStream(ctx, streamName, readOpts, count)
	if err != nil {
		log.Printf("Error reading stream: %v", err)
		return nil, err
	}
	defer stream.Close()

	// Iterate over the events using Recv()
	for {
		// Receive the next event
		event, err := stream.Recv()
		if err != nil {
			// If we get an error, handle it (e.g., no more events or another error)
			if err.Error() == "EOF" { // End of stream
				break
			}
			log.Printf("Error receiving event: %v", err)
			return nil, err
		}

		// Unmarshal the event data into the model.Event struct
		var e model.Event
		if err := json.Unmarshal(event.Event.Data, &e); err != nil {
			log.Printf("Error unmarshalling event data: %v", err)
			continue
		}

		// Append the event to the events slice
		events = append(events, e)
	}

	return events, nil
}
