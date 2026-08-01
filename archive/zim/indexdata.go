package zim

// IndexData provides metadata needed for full-text indexing of a ZIM item.
// Items that implement this interface will have their content indexed
// when the Writer is configured with SetIndexing(true, language).
type IndexData interface {
	// HasIndexData returns true if this item should be indexed.
	HasIndexData() bool

	// GetTitle returns the title for indexing (may differ from the entry title).
	GetTitle() string

	// GetContent returns the plain text content for full-text indexing.
	GetContent() string

	// GetKeywords returns additional keywords to boost in search results.
	GetKeywords() string

	// GetWordCount returns the number of words in the content.
	GetWordCount() uint64

	// GetGeoPosition returns the geographic position of this item, if any.
	// ok is false if no geo position is available.
	GetGeoPosition() (latitude, longitude float64, ok bool)
}

// NoIndexData is a default IndexData implementation that disables indexing.
type NoIndexData struct{}

func (n NoIndexData) HasIndexData() bool                        { return false }
func (n NoIndexData) GetTitle() string                          { return "" }
func (n NoIndexData) GetContent() string                        { return "" }
func (n NoIndexData) GetKeywords() string                       { return "" }
func (n NoIndexData) GetWordCount() uint64                      { return 0 }
func (n NoIndexData) GetGeoPosition() (float64, float64, bool)  { return 0, 0, false }

// SimpleIndexData provides basic index data from a title, content, and keywords.
type SimpleIndexData struct {
	title       string
	content     string
	keywords    string
	wordCount   uint64
	geoPosition struct {
		lat, lon float64
		has      bool
	}
}

// NewSimpleIndexData creates an IndexData with the given title and content.
func NewSimpleIndexData(title, content string) *SimpleIndexData {
	return &SimpleIndexData{title: title, content: content}
}

func (s *SimpleIndexData) HasIndexData() bool { return true }

func (s *SimpleIndexData) GetTitle() string    { return s.title }
func (s *SimpleIndexData) GetContent() string  { return s.content }
func (s *SimpleIndexData) GetKeywords() string { return s.keywords }
func (s *SimpleIndexData) GetWordCount() uint64 { return s.wordCount }

func (s *SimpleIndexData) GetGeoPosition() (float64, float64, bool) {
	return s.geoPosition.lat, s.geoPosition.lon, s.geoPosition.has
}

// SetKeywords sets the keywords for this index data.
func (s *SimpleIndexData) SetKeywords(keywords string) {
	s.keywords = keywords
}

// SetWordCount sets the word count for this index data.
func (s *SimpleIndexData) SetWordCount(count uint64) {
	s.wordCount = count
}

// SetGeoPosition sets the geographic position for this index data.
func (s *SimpleIndexData) SetGeoPosition(latitude, longitude float64) {
	s.geoPosition.lat = latitude
	s.geoPosition.lon = longitude
	s.geoPosition.has = true
}
