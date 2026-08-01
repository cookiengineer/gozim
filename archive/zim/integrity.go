package zim

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// CheckType specifies which integrity checks to perform.
type CheckType uint32

const (
	CheckChecksum        CheckType = 1 << iota
	CheckDirentPtrs
	CheckDirentOrder
	CheckTitleIndex
	CheckClusterPtrs
	CheckClusterOffsets
	CheckMimeTypes
	CheckRedirectLoops
	CheckLinks
	CheckRedundancy
	CheckEmptyEntries
	CheckAll CheckType = (1 << iota) - 1
)

// CheckReport holds the results of integrity checks.
type CheckReport struct {
	ChecksumOK    bool
	BrokenLinks   []BrokenLink
	RedirectLoops [][]uint32
	Redundant     []RedundantPair
	Errors        []string
}

// BrokenLink describes an internal link that points to a missing entry.
type BrokenLink struct {
	SourceEntry string
	TargetLink  string
	Reason      string
}

// RedundantPair describes two entries with identical content.
type RedundantPair struct {
	EntryA uint32
	EntryB uint32
}

// CheckFile performs integrity checks on a ZIM file.
func CheckFile(path string, checks CheckType) (*CheckReport, error) {
	archive, err := Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening archive: %w", err)
	}
	defer archive.Close()

	report := &CheckReport{ChecksumOK: true}

	if checks&CheckChecksum != 0 && archive.HasChecksum() {
		if err := VerifyChecksum(archive.reader, archive.header.ChecksumPos); err != nil {
			report.ChecksumOK = false
			report.Errors = append(report.Errors, err.Error())
		}
	}

	if checks&CheckDirentPtrs != 0 {
		report.checkDirentPtrs(archive)
	}

	if checks&CheckDirentOrder != 0 {
		report.checkDirentOrder(archive)
	}

	if checks&CheckMimeTypes != 0 {
		report.checkMimeTypes(archive)
	}

	if checks&CheckRedirectLoops != 0 {
		report.checkRedirectLoops(archive)
	}

	if checks&CheckLinks != 0 {
		report.checkLinks(archive)
	}

	if checks&CheckEmptyEntries != 0 {
		report.checkEmptyEntries(archive)
	}

	return report, nil
}

func (r *CheckReport) checkDirentPtrs(a *Archive) {
	for i, offset := range a.urlPtrs {
		if offset >= uint64(a.reader.Size()) {
			r.Errors = append(r.Errors, fmt.Sprintf("dirent %d: offset %d exceeds file size", i, offset))
		}
		if i > 0 && offset <= a.urlPtrs[i-1] {
			r.Errors = append(r.Errors, fmt.Sprintf("dirent %d: offset %d not monotonically increasing", i, offset))
		}
	}
}

func (r *CheckReport) checkDirentOrder(a *Archive) {
	for i := 1; i < len(a.dirents); i++ {
		if a.dirents[i].Path < a.dirents[i-1].Path {
			r.Errors = append(r.Errors, fmt.Sprintf("dirent %d: paths out of order (%q < %q)", i, a.dirents[i].Path, a.dirents[i-1].Path))
		}
	}
}

func (r *CheckReport) checkMimeTypes(a *Archive) {
	for i, dirent := range a.dirents {
		if err := a.mimeList.Validate(dirent.MimeTypeIndex); err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("dirent %d: %v", i, err))
		}
	}
}

func (r *CheckReport) checkRedirectLoops(a *Archive) {
	visited := make(map[uint32]int) // index → visit round

	for i := range a.dirents {
		if !a.dirents[i].IsRedirect() {
			continue
		}

		slow := uint32(i)
		fast := uint32(i)
		round := 0

		for {
			if int(fast) >= len(a.dirents) || !a.dirents[fast].IsRedirect() {
				break
			}
			fast = a.dirents[fast].RedirectIndex
			if round%2 == 1 {
				if int(slow) >= len(a.dirents) || !a.dirents[slow].IsRedirect() {
					break
				}
				slow = a.dirents[slow].RedirectIndex
			}
			if fast == slow && round > 0 {
				// Found a cycle.
				var loop []uint32
				loop = append(loop, fast)
				curr := a.dirents[fast].RedirectIndex
				for curr != fast {
					loop = append(loop, curr)
					curr = a.dirents[curr].RedirectIndex
				}
				r.RedirectLoops = append(r.RedirectLoops, loop)
				break
			}
			round++
		}

		_ = visited
	}
}

func (r *CheckReport) checkLinks(a *Archive) {
	for _, dirent := range a.dirents {
		if dirent.IsRedirect() || dirent.IsDeleted() {
			continue
		}

		mimeType := a.mimeList.MimeType(dirent.MimeTypeIndex)
		if !strings.HasPrefix(mimeType, "text/html") {
			continue
		}

		// Read the HTML content and check internal links.
		item, err := a.readItem(dirent)
		if err != nil {
			continue
		}

		itemData, itemErr := item.DataAll()
		if itemErr != nil {
			continue
		}
		content := string(itemData)

		// Simple HTML link extraction.
		for _, attr := range []string{"href=", "src="} {
			pos := 0
			for {
				idx := strings.Index(content[pos:], attr)
				if idx < 0 {
					break
				}
				pos += idx + len(attr)

				// Skip external URLs.
				rest := content[pos:]
				if len(rest) < 2 {
					break
				}

				var quote byte
				if rest[0] == '"' || rest[0] == '\'' {
					quote = rest[0]
					rest = rest[1:]
				} else {
					// Unquoted attribute.
					spaceIdx := strings.IndexAny(rest, " \t\n\r>")
					if spaceIdx < 0 {
						spaceIdx = len(rest)
					}
					link := rest[:spaceIdx]
					pos += spaceIdx

					if !isExternalURL(link) && !a.HasEntry(link) {
						r.BrokenLinks = append(r.BrokenLinks, BrokenLink{
							SourceEntry: dirent.Path,
							TargetLink:  link,
							Reason:      "entry not found",
						})
					}
					continue
				}

				if quote == 0 {
					continue
				}
				closingIdx := strings.IndexByte(rest, quote)
				if closingIdx < 0 {
					break
				}

				link := rest[:closingIdx]
				pos += closingIdx + 1

				if !isExternalURL(link) && link != "" && !a.HasEntry(link) {
					r.BrokenLinks = append(r.BrokenLinks, BrokenLink{
						SourceEntry: dirent.Path,
						TargetLink:  link,
						Reason:      "entry not found",
					})
				}
			}
		}
	}
}

func (r *CheckReport) checkEmptyEntries(a *Archive) {
	for i, dirent := range a.dirents {
		if dirent.IsContent() {
			item, err := a.readItem(dirent)
			if err != nil {
				continue
			}
			if item.Size() == 0 {
				r.Errors = append(r.Errors, fmt.Sprintf("dirent %d (%q): empty content", i, dirent.Path))
			}
		}
	}
}

func isExternalURL(url string) bool {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return true
	}
	if strings.HasPrefix(url, "mailto:") || strings.HasPrefix(url, "tel:") {
		return true
	}
	if strings.HasPrefix(url, "javascript:") || strings.HasPrefix(url, "data:") {
		return true
	}
	if strings.HasPrefix(url, "//") {
		return true
	}
	if strings.HasPrefix(url, "#") {
		return true
	}
	return false
}

// checkRedundancy finds entries with identical content.
func (r *CheckReport) checkRedundancy(a *Archive) {
	hashes := make(map[string]uint32) // hex hash → first entry index

	for i, dirent := range a.dirents {
		if !dirent.IsContent() {
			continue
		}

		item, err := a.readItem(dirent)
		if err != nil {
			continue
		}

		data, itemErr := item.DataAll()
		if itemErr != nil {
			continue
		}
		hash := fmt.Sprintf("%x", md5Hash(data))

		if firstIdx, ok := hashes[hash]; ok {
			r.Redundant = append(r.Redundant, RedundantPair{
				EntryA: firstIdx,
				EntryB: uint32(i),
			})
		} else {
			hashes[hash] = uint32(i)
		}
	}
}

func md5Hash(data []byte) [16]byte {
	return md5.Sum(data)
}
