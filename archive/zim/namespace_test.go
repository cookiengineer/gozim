package zim

import (
	"testing"
)

func TestNamespaceIsUserContent(t *testing.T) {
	tests := []struct {
		ns   Namespace
		want bool
	}{
		{NamespaceContent, true},
		{NamespaceArticle, true},
		{NamespaceImage, true},
		{NamespaceScript, true},
		{NamespaceLayout, true},
		{NamespaceMetadata, false},
		{NamespaceIndex, false},
		{Namespace('W'), false},
		{Namespace('Z'), false},
		{Namespace(0), false},
	}

	for _, tt := range tests {
		t.Run(string(byte(tt.ns)), func(t *testing.T) {
			if got := tt.ns.IsUserContent(); got != tt.want {
				t.Errorf("IsUserContent(%q) = %v, want %v", tt.ns, got, tt.want)
			}
		})
	}
}

func TestNamespaceIsMetadata(t *testing.T) {
	tests := []struct {
		ns   Namespace
		want bool
	}{
		{NamespaceMetadata, true},
		{NamespaceContent, false},
		{NamespaceArticle, false},
		{NamespaceIndex, false},
	}

	for _, tt := range tests {
		if got := tt.ns.IsMetadata(); got != tt.want {
			t.Errorf("IsMetadata(%q) = %v, want %v", string(byte(tt.ns)), got, tt.want)
		}
	}
}

func TestNamespaceIsIndex(t *testing.T) {
	tests := []struct {
		ns   Namespace
		want bool
	}{
		{NamespaceIndex, true},
		{NamespaceContent, false},
		{NamespaceMetadata, false},
	}

	for _, tt := range tests {
		if got := tt.ns.IsIndex(); got != tt.want {
			t.Errorf("IsIndex(%q) = %v, want %v", string(byte(tt.ns)), got, tt.want)
		}
	}
}

func TestNamespaceString(t *testing.T) {
	tests := []struct {
		ns   Namespace
		want string
	}{
		{NamespaceContent, "C"},
		{NamespaceMetadata, "M"},
		{NamespaceIndex, "X"},
		{NamespaceArticle, "A"},
		{NamespaceImage, "I"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.ns.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
