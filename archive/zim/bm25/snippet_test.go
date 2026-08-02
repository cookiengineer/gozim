package bm25

import (
	"strings"
	"testing"
)

func TestGenerateSnippetEmptyContent(t *testing.T) {
	result := GenerateSnippet("", []string{"test"}, 200)
	if result != "" {
		t.Errorf("expected empty snippet, got %q", result)
	}
}

func TestGenerateSnippetEmptyTerms(t *testing.T) {
	result := GenerateSnippet("hello world", nil, 200)
	if !strings.Contains(result, "hello") {
		t.Errorf("expected content in snippet, got %q", result)
	}
}

func TestGenerateSnippetNoMatch(t *testing.T) {
	result := GenerateSnippet("hello world", []string{"xyz"}, 200)
	if !strings.Contains(result, "hello") {
		t.Errorf("expected content in snippet, got %q", result)
	}
}

func TestGenerateSnippetBasicHighlight(t *testing.T) {
	result := GenerateSnippet("The quick brown fox jumps over the lazy dog", []string{"fox"}, 200)
	if !strings.Contains(result, "**fox**") {
		t.Errorf("expected **fox** in snippet, got %q", result)
	}
}

func TestGenerateSnippetMultiTermHighlight(t *testing.T) {
	result := GenerateSnippet("The quick brown fox jumps over the lazy dog", []string{"fox", "dog"}, 200)
	if !strings.Contains(result, "**fox**") {
		t.Errorf("expected **fox** in snippet, got %q", result)
	}
	if !strings.Contains(result, "**dog**") {
		t.Errorf("expected **dog** in snippet, got %q", result)
	}
}

func TestGenerateSnippetCaseInsensitive(t *testing.T) {
	result := GenerateSnippet("The Quick Brown Fox", []string{"fox"}, 200)
	if !strings.Contains(result, "**Fox**") {
		t.Errorf("expected **Fox** (preserving original case) in snippet, got %q", result)
	}
}

func TestGenerateSnippetTruncation(t *testing.T) {
	content := "short padding text here sigma value extra text after here"
	result := GenerateSnippet(content, []string{"sigma"}, 30)
	if !strings.Contains(result, "**sigma**") {
		t.Errorf("expected **sigma** in snippet, got %q", result)
	}
}

func TestGenerateSnippetOverlappingTerms(t *testing.T) {
	result := GenerateSnippet("abcde abcde", []string{"ab", "bc"}, 200)
	if !strings.Contains(result, "**ab**") {
		t.Errorf("expected **ab** in snippet, got %q", result)
	}
}

func TestGenerateSnippetZeroMaxLen(t *testing.T) {
	result := GenerateSnippet("hello world", []string{"world"}, 0)
	if !strings.Contains(result, "**world**") {
		t.Errorf("expected highlighted term with default maxLen, got %q", result)
	}
}

func TestGenerateSnippetNegativeMaxLen(t *testing.T) {
	result := GenerateSnippet("hello world", []string{"world"}, -1)
	if !strings.Contains(result, "**world**") {
		t.Errorf("expected highlighted term with default maxLen, got %q", result)
	}
}

func TestGenerateSnippetContentSmallerThanMaxLen(t *testing.T) {
	result := GenerateSnippet("hello", []string{"hello"}, 200)
	if !strings.Contains(result, "**hello**") {
		t.Errorf("expected highlighted short content, got %q", result)
	}
}

func TestGenerateSnippetMultipleMatchesSameTerm(t *testing.T) {
	content := "foo bar baz foo bar baz foo bar baz"
	result := GenerateSnippet(content, []string{"foo"}, 200)
	count := strings.Count(result, "**foo**")
	if count == 0 {
		t.Errorf("expected at least one highlighted foo, got %q", result)
	}
}

func TestGenerateSnippetWithHTMLContent(t *testing.T) {
	content := `<html><body><p>type Reader</p><a href="#Reader">link</a></body></html>`
	result := GenerateSnippet(content, []string{"Reader"}, 200)
	if !strings.Contains(result, "**Reader**") {
		t.Errorf("expected **Reader** in snippet, got %q", result)
	}
}

func TestGenerateSnippetQueryTermsOrderPreservesOriginalCase(t *testing.T) {
	result := GenerateSnippet("FOO bar BAZ", []string{"foo"}, 200)
	if strings.Contains(result, "**foo**") {
		t.Errorf("expected **FOO** (preserving original case), got %q", result)
	}
	if !strings.Contains(result, "**FOO**") {
		t.Errorf("expected **FOO** in snippet, got %q", result)
	}
}

func TestGenerateSnippetSecondTermDoesNotCorrupt(t *testing.T) {
	content := "The io.Reader and io.Writer interfaces are fundamental to Go's I/O model. The Reader defines a single method."
	result := GenerateSnippet(content, []string{"Reader", "Writer"}, 200)
	if !strings.Contains(result, "**Reader**") {
		t.Errorf("expected **Reader** in snippet, got %q", result)
	}
	if !strings.Contains(result, "**Writer**") {
		t.Errorf("expected **Writer** in snippet, got %q", result)
	}
	if strings.Contains(result, "****") {
		t.Errorf("unexpected nested bold markers in snippet: %q", result)
	}
}

func TestGenerateSnippetGoTypePointerBug(t *testing.T) {
	content := "type Reader struct{}\nfunc NewReader(rd io.Reader) *Reader"
	result := GenerateSnippet(content, []string{"Reader"}, 100)
	if strings.Count(result, "**")%2 != 0 {
		t.Errorf("unbalanced ** markers in snippet: %q", result)
	}
}

func TestGenerateSnippetEllipsisPrefix(t *testing.T) {
	content := "aaa bbb ccc ddd eee fff ggg hhh iii jjj kkk lll mmm nnn ooo ppp qqq rrr sss ttt uuu vvv www xxx yyy zzz"
	result := GenerateSnippet(content, []string{"zzz"}, 10)
	if !strings.HasPrefix(result, "...") {
		t.Errorf("expected '...' prefix when best match is far from start, got %q", result)
	}
}

func TestGenerateSnippetEllipsisSuffix(t *testing.T) {
	content := "aaa bbb ccc ddd eee fff ggg hhh iii jjj kkk lll mmm nnn ooo ppp qqq rrr sss ttt uuu vvv www xxx yyy zzz"
	result := GenerateSnippet(content, []string{"aaa"}, 10)
	if !strings.HasSuffix(result, "...") {
		t.Errorf("expected '...' suffix when content extends beyond window, got %q", result)
	}
}

func TestGenerateSnippetNoSliceBoundsPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GenerateSnippet panicked: %v", r)
		}
	}()

	content := `<ul><li><a href="#Reader">type <b>Reader</b></a></li><li><a href="#NewReaderSize">func NewReaderSize(rd io.Reader, size int) *Reader</a></li></ul>`
	_ = GenerateSnippet(content, []string{"Reader"}, 200)
}
