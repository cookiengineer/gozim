package bm25

import (
	"reflect"
	"testing"
)

func TestParseQueryEmpty(t *testing.T) {
	terms, phrases, notTerms := ParseQuery("")
	if len(terms) != 0 || len(phrases) != 0 || len(notTerms) != 0 {
		t.Errorf("expected all empty, got terms=%v phrases=%v notTerms=%v", terms, phrases, notTerms)
	}
}

func TestParseQuerySimpleTerms(t *testing.T) {
	terms, phrases, notTerms := ParseQuery("hello world")
	expected := []string{"hello", "world"}
	if !reflect.DeepEqual(terms, expected) {
		t.Errorf("expected %v, got %v", expected, terms)
	}
	if len(phrases) != 0 || len(notTerms) != 0 {
		t.Errorf("expected no phrases or notTerms, got phrases=%v notTerms=%v", phrases, notTerms)
	}
}

func TestParseQueryNOTTerm(t *testing.T) {
	terms, _, notTerms := ParseQuery("golang -python -ruby")
	if !reflect.DeepEqual(terms, []string{"golang"}) {
		t.Errorf("expected [golang], got %v", terms)
	}
	if !reflect.DeepEqual(notTerms, []string{"python", "ruby"}) {
		t.Errorf("expected [python ruby], got %v", notTerms)
	}
}

func TestParseQueryNOTTermNormalized(t *testing.T) {
	terms, _, notTerms := ParseQuery("hello -WORLD")
	if !reflect.DeepEqual(terms, []string{"hello"}) {
		t.Errorf("expected [hello], got %v", terms)
	}
	if !reflect.DeepEqual(notTerms, []string{"world"}) {
		t.Errorf("expected [world] (lowercased), got %v", notTerms)
	}
}

func TestParseQueryMultipleSpaces(t *testing.T) {
	terms, _, _ := ParseQuery("  hello   world  ")
	if !reflect.DeepEqual(terms, []string{"hello", "world"}) {
		t.Errorf("expected [hello world], got %v", terms)
	}
}

func TestParseQueryMixedCase(t *testing.T) {
	terms, _, _ := ParseQuery("Hello WORLD")
	if !reflect.DeepEqual(terms, []string{"hello", "world"}) {
		t.Errorf("expected [hello world], got %v", terms)
	}
}

func TestParseQueryTrailingPunctuation(t *testing.T) {
	terms, _, _ := ParseQuery("hello. world! test?")
	if !reflect.DeepEqual(terms, []string{"hello", "world", "test"}) {
		t.Errorf("expected [hello world test], got %v", terms)
	}
}

func TestParseQueryDoubleDashNotStripped(t *testing.T) {
	terms, _, notTerms := ParseQuery("--flag")
	if !reflect.DeepEqual(terms, []string{"flag"}) {
		t.Errorf("expected [flag] (-- stripped by normalizeTerm), got %v", terms)
	}
	if len(notTerms) != 0 {
		t.Errorf("expected no notTerms for --flag, got %v", notTerms)
	}
}

func TestNormalizeTermLowercase(t *testing.T) {
	result := normalizeTerm("HELLO")
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestNormalizeTermStripPunctuation(t *testing.T) {
	result := normalizeTerm("hello!")
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestNormalizeTermEmpty(t *testing.T) {
	result := normalizeTerm("")
	if result != "" {
		t.Errorf("expected '', got %q", result)
	}
}

func TestNormalizeTermOnlyPunctuation(t *testing.T) {
	result := normalizeTerm("!")
	if result != "" {
		t.Errorf("expected '', got %q", result)
	}
}
