package event_network

import "time"

const (
	// domains
	Geology           = "geology"
	AnimalObservation = "animal_observation"
	// events: geology
	MinorTremors = "minor_tremors"

	// events: AnimalObservation
	ZebrasMigration                  = "zebras_migration"
	UnusualBirdBehavior              = "unusual_bird_behavior"
	MultipleAnimalUnexpectedBehavior = "multiple_animal_unexpected_behavior"
)

var animalUnexpectedBehavior = time.Date(2026, 25, 4, 13, 3, 0, 0, time.UTC)

func createZebrasEvent() Event {
	event := Event{
		EventType:   ZebrasMigration,
		EventDomain: AnimalObservation,
		Timestamp:   time.Date(2026, 25, 4, 5, 3, 0, 0, time.UTC),
	}
	return event
}

func createUnusualBirdBehaviorEvent() Event {
	event := Event{
		EventType:   UnusualBirdBehavior,
		EventDomain: AnimalObservation,
		Timestamp:   animalUnexpectedBehavior,
	}
	return event
}

func getAnimalObservationDerivedEventTemplate() EventTemplate {
	return EventTemplate{
		EventType:   MultipleAnimalUnexpectedBehavior,
		EventDomain: AnimalObservation,
	}
}

func getMinorTremorDerivedEventTemplate() EventTemplate {
	return EventTemplate{
		EventType:   MinorTremors,
		EventDomain: Geology,
	}
}
