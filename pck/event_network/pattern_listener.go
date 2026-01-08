package event_network

import (
	"encoding/json"
	"fmt"
)

// PatternListener is  “fire event or method call” sink.
type PatternListener interface {
	OnPatternRepeated(match PatternMatch)
}

func NewPatternListenerPoc() *PatternListenerPoc {
	return &PatternListenerPoc{}
}

type PatternListenerPoc struct {
}

func (p *PatternListenerPoc) OnPatternRepeated(match PatternMatch) {
	str, _ := json.MarshalIndent(match, "", "	")
	fmt.Println("PATTERN REPEATED --------------")
	fmt.Println(string(str))
	fmt.Println("-------------------------------", match.Occurrence)
}
