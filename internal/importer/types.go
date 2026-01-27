package importer

// YAMLGovernmentFundingPeriod represents a period in the YAML government funding file.
type YAMLGovernmentFundingPeriod struct {
	From    string                       `yaml:"from"`
	To      string                       `yaml:"to"`
	Comment string                       `yaml:"comment,omitempty"`
	Entries []YAMLGovernmentFundingEntry `yaml:"entries"`
}

// YAMLGovernmentFundingEntry represents an age-based entry in the YAML government funding file.
type YAMLGovernmentFundingEntry struct {
	Age        [2]int                                   `yaml:"age"` // [min, max]
	Properties map[string]YAMLGovernmentFundingProperty `yaml:"properties"`
}

// YAMLGovernmentFundingProperty represents a property with payment and requirement.
type YAMLGovernmentFundingProperty struct {
	Payment     float64 `yaml:"payment"` // EUR amount (converted to cents on import)
	Requirement float64 `yaml:"requirement"`
	Comment     string  `yaml:"comment,omitempty"`
}
