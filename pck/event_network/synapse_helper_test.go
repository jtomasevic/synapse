package event_network

import "time"

const (
	// domains
	NaturalDisasterWarningSystem = "natural_disaster_warning"
	Geology                      = "geology"
	AnimalObservation            = "animal_observation"
	// events: geology
	MinorTremors = "minor_tremors"
	// derive event:
	HighFrequencyOfMinorTremors = "high_frequency_of_minor_tremors"
	// events: AnimalObservation
	ZebrasMigration     = "zebras_migration"
	UnusualBirdBehavior = "unusual_bird_behavior"
	// derive event
	MultipleAnimalUnexpectedBehavior = "multiple_animal_unexpected_behavior"

	// cross domain derived event
	PotentialNaturalCatastrophic = "potential_natural_catastrophic"
)

var animalUnexpectedBehavior = time.Date(2026, 4, 25, 13, 3, 0, 0, time.UTC)

func createZebrasEvent() Event {
	event := Event{
		EventType:   ZebrasMigration,
		EventDomain: AnimalObservation,
		Timestamp:   time.Date(2026, 4, 25, 5, 3, 0, 0, time.UTC),
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

func getPotentialNaturalCatastrophicDerivedEventTemplate() EventTemplate {
	return EventTemplate{
		EventType:   PotentialNaturalCatastrophic,
		EventDomain: NaturalDisasterWarningSystem,
	}
}

func createMinorTremorsEvent(timestamp time.Time) Event {
	return Event{
		EventType:   MinorTremors,
		EventDomain: Geology,
		Timestamp:   timestamp,
	}
}

func getMinorTremorsEvents() []Event {
	return []Event{
		createMinorTremorsEvent(
			time.Date(2026, 4, 25, 5, 11, 0, 0, time.UTC),
		),
		createMinorTremorsEvent(
			time.Date(2026, 4, 25, 6, 17, 0, 0, time.UTC),
		),
		createMinorTremorsEvent(
			time.Date(2026, 4, 2, 6, 18, 0, 0, time.UTC),
		),
		createMinorTremorsEvent(
			time.Date(2026, 4, 25, 6, 44, 0, 0, time.UTC),
		),
		createMinorTremorsEvent(
			time.Date(2026, 4, 25, 7, 21, 0, 0, time.UTC),
		),
		createMinorTremorsEvent(
			time.Date(2026, 4, 25, 7, 5, 0, 0, time.UTC),
		),
		createMinorTremorsEvent(
			time.Date(2026, 4, 25, 12, 23, 0, 0, time.UTC),
		),
		createMinorTremorsEvent(
			time.Date(2026, 4, 25, 12, 53, 0, 0, time.UTC),
		),
	}
}

func getMinorTremorDerivedEventTemplate() EventTemplate {
	return EventTemplate{
		EventType:   HighFrequencyOfMinorTremors,
		EventDomain: Geology,
	}
}
