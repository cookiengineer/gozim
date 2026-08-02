package utils

import "strings"

func DetectMimeType(filename string) string {
	ext := strings.ToLower(filename[strings.LastIndex(filename, ".")+1:])
	switch ext {
	case "html", "htm":
		return "text/html"
	case "css":
		return "text/css"
	case "js":
		return "application/javascript"
	case "json":
		return "application/json"
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "svg":
		return "image/svg+xml"
	case "xml":
		return "application/xml"
	case "pdf":
		return "application/pdf"
	case "txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
