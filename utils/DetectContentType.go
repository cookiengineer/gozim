package utils

import (
	"fmt"
)

func DetectContentType(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	switch {
	case data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G':
		return "[PNG]"
	case data[0] == 0xFF && data[1] == 0xD8:
		return "[JPEG]"
	case data[0] == 'G' && data[1] == 'I' && data[2] == 'F':
		return "[GIF]"
	case len(data) >= 9 && string(data[:9]) == "<!DOCTYPE":
		return "[HTML]"
	case data[0] == '<' && len(data) >= 5 && (data[1] == 'h' || data[1] == 'H'):
		return "[HTML]"
	case data[0] == 0x1F && data[1] == 0x8B:
		return "[gzip]"
	case data[0] == 0x28 && data[1] == 0xB5 && data[2] == 0x2F && data[3] == 0xFD:
		return "[zstd]"
	default:
		return fmt.Sprintf("[%02X %02X %02X %02X]", data[0], data[1], data[2], data[3])
	}
}

