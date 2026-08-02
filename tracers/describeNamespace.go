package tracers

import (
	"fmt"
	"github.com/cookiengineer/gozim/archive/zim"
)

func describeNamespace(ns zim.Namespace) string {
	switch ns {
	case zim.NamespaceContent:
		return "content (new scheme)"
	case zim.NamespaceArticle:
		return "article (old scheme)"
	case zim.NamespaceImage:
		return "image (old scheme)"
	case zim.NamespaceScript:
		return "script (old scheme)"
	case zim.NamespaceLayout:
		return "layout (old scheme)"
	case zim.NamespaceMetadata:
		return "metadata"
	case zim.NamespaceIndex:
		return "index"
	case 'W':
		return "listing/redirect"
	default:
		if ns >= 'A' && ns <= 'Z' {
			return fmt.Sprintf("namespace %c", ns)
		}
		return fmt.Sprintf("unknown (0x%02X)", byte(ns))
	}
}

