package sources

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	stiftungListURL  = "https://www.kitabern.ch/kindertagesstaetten"
	stiftungProvider = "Stiftung Kindertagesstätten Bern"
)

// stiftungEntry is the list-level info parsed from the overview page.
type stiftungEntry struct {
	Name      string // e.g. "Kita Taka Tuka"
	Location  string // e.g. "3012 Bern" (raw)
	URL       string // external Kita website
	PhotoURL  string
}

// Address/phone patterns
var (
	// e.g. "Länggassstrasse 64" (street + house number)
	streetRe = regexp.MustCompile(`^\s*([A-ZÄÖÜ][\w\-äöüéèà]+(?:strasse|weg|gasse|platz|allee|rain|ring|hof)\s+\d+[A-Za-z]?)\s*$`)
	// e.g. "3012 Bern"
	plzCityRe = regexp.MustCompile(`\b(\d{4})\s+([A-ZÄÖÜ][\w\-]+)\b`)
	// Obfuscated email like "info(a)kita-foo.ch" — strict TLD to avoid trailing text bleed.
	obfuscatedEmailRe = regexp.MustCompile(`([A-Za-z0-9][\w\.\-]*)\(a\)([A-Za-z0-9][\w\-]*(?:\.[A-Za-z0-9][\w\-]*)*\.[a-z]{2,6})\b`)
)

// ScrapeStiftung fetches all Kitas of Stiftung Kindertagesstätten Bern (kitabern.ch).
// Each Kita lives on its own external domain, so we fetch each site's homepage
// and /kontakt page and extract contact data heuristically.
func ScrapeStiftung(outputPath string) error {
	entries, err := stiftungList()
	if err != nil {
		return fmt.Errorf("fetch kita list: %w", err)
	}
	log.Printf("Found %d Kitas", len(entries))

	var kitas []kitaData
	for _, e := range entries {
		k := stiftungDetail(e)
		log.Printf("  %s — phone=%q email=%q addr=%q leitung=%q",
			k.Name, k.Phone, k.Email, k.Address, k.LeitungName)
		kitas = append(kitas, k)
	}

	return writeExcel(outputPath, stiftungProvider, kitas)
}

// stiftungList parses the overview page to extract Kita name, location and external URL.
func stiftungList() ([]stiftungEntry, error) {
	doc, err := fetch(stiftungListURL)
	if err != nil {
		return nil, err
	}

	var entries []stiftungEntry
	seen := map[string]bool{}

	doc.Find("h2 a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		href = strings.TrimSpace(href)
		if !strings.HasPrefix(href, "http") || seen[href] {
			return
		}
		// The overview page anchors also include social/footer links - filter by text shape.
		txt := normalizeWhitespace(s.Text())
		if txt == "" || !strings.Contains(strings.ToLower(txt), "kita") {
			return
		}
		name, location := splitNameLocation(txt)
		seen[href] = true
		entries = append(entries, stiftungEntry{
			Name:     name,
			Location: location,
			URL:      href,
		})
	})

	// Attach first content image near each anchor as PhotoURL (best effort).
	doc.Find("div.ce-textpic img[src]").Each(func(_ int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		if src == "" {
			return
		}
		// Walk up to find the enclosing section, then look for its preceding header link.
		section := s.ParentsFiltered("article, section, .kita, div.csc-default").First()
		if section.Length() == 0 {
			return
		}
		href := section.Find("h2 a[href]").First().AttrOr("href", "")
		if href == "" {
			return
		}
		for i := range entries {
			if entries[i].URL == href && entries[i].PhotoURL == "" {
				entries[i].PhotoURL = absoluteURL(stiftungListURL, src)
				return
			}
		}
	})

	return entries, nil
}

// splitNameLocation splits "Kita Taka Tuka, 3012 Bern" → ("Kita Taka Tuka", "3012 Bern").
func splitNameLocation(s string) (name, location string) {
	i := strings.LastIndex(s, ",")
	if i < 0 {
		return strings.TrimSpace(s), ""
	}
	return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
}

// stiftungDetail scrapes one Kita by fetching its homepage and /kontakt page.
func stiftungDetail(e stiftungEntry) kitaData {
	k := kitaData{
		Name:     e.Name,
		PhotoURL: e.PhotoURL,
	}

	// Try /kontakt first — Betriebsleitung block is usually there.
	kontakt := strings.TrimRight(e.URL, "/") + "/kontakt"
	if doc, err := fetch(kontakt); err == nil {
		enrichFromDoc(doc, &k)
	}

	// Fallback to homepage for any still-missing fields.
	if k.Phone == "" || k.Email == "" || k.Address == "" {
		if doc, err := fetch(e.URL); err == nil {
			enrichFromDoc(doc, &k)
		}
	}

	// If location wasn't populated from address, synthesize from list entry.
	if k.Address == "" && e.Location != "" {
		k.Address = e.Location
	}

	// Derive a URL note so imported records link back to the Kita.
	if k.Notes == "" {
		k.Notes = e.URL
	}

	return k
}

// enrichFromDoc fills any empty contact field in k from the parsed document.
func enrichFromDoc(doc *goquery.Document, k *kitaData) {
	// Phone: first tel: link (normalize text to the formatted visible version).
	if k.Phone == "" {
		doc.Find("a[href^='tel:']").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			if txt := strings.TrimSpace(s.Text()); txt != "" {
				k.Phone = normalizeWhitespace(txt)
				return false
			}
			href, _ := s.Attr("href")
			k.Phone = strings.TrimPrefix(href, "tel:")
			return false
		})
	}

	// Email: prefer real mailto:, else decode obfuscated "(a)" pattern from anchor text.
	if k.Email == "" {
		doc.Find("a[href^='mailto:']").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			href, _ := s.Attr("href")
			addr := strings.TrimPrefix(href, "mailto:")
			if strings.Contains(addr, "@") {
				k.Email = addr
				return false
			}
			return true
		})
	}
	if k.Email == "" {
		// Obfuscated case: <a href="#" data-mailto-token="...">info(a)domain.ch</a>.
		// Scan each anchor's own text so we don't pick up neighbouring characters.
		doc.Find("a").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			txt := strings.TrimSpace(s.Text())
			if !strings.Contains(txt, "(a)") && !strings.Contains(txt, "[at]") {
				return true
			}
			if m := obfuscatedEmailRe.FindStringSubmatch(txt); len(m) == 3 {
				k.Email = m[1] + "@" + m[2]
				return false
			}
			return true
		})
	}

	// Leitung + Address: look for "Betriebsleitung" / "Leitung" label,
	// then parse the following paragraph.
	if k.LeitungName == "" || k.Address == "" {
		doc.Find("h1, h2, h3, h4, strong, b").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			label := strings.ToLower(strings.TrimSpace(s.Text()))
			if !strings.Contains(label, "leitung") || strings.Contains(label, "administration") {
				return true
			}
			// Walk to the next <p> sibling in document order.
			p := nextParagraph(s)
			if p == nil {
				return true
			}
			lines := paragraphLines(p)
			if len(lines) == 0 {
				return true
			}
			if k.LeitungName == "" && looksLikePersonName(lines[0]) {
				k.LeitungName = lines[0]
			}
			if k.Address == "" {
				k.Address = extractAddress(lines)
			}
			return false
		})
	}

	// Last-resort address extraction: scan all paragraphs for street + PLZ.
	if k.Address == "" {
		doc.Find("p, address, li").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			lines := paragraphLines(s)
			if addr := extractAddress(lines); addr != "" {
				k.Address = addr
				return false
			}
			return true
		})
	}
}

// nextParagraph returns the first <p> encountered walking forward from s.
func nextParagraph(s *goquery.Selection) *goquery.Selection {
	cur := s
	for range [6]int{} {
		next := cur.Next()
		if next.Length() == 0 {
			cur = cur.Parent()
			if cur.Length() == 0 {
				return nil
			}
			continue
		}
		if next.Is("p") {
			return next
		}
		if p := next.Find("p").First(); p.Length() > 0 {
			return p
		}
		cur = next
	}
	return nil
}

// paragraphLines splits a <p>'s inner text on <br> and newlines, trimming blanks.
func paragraphLines(p *goquery.Selection) []string {
	html, err := p.Html()
	if err != nil {
		return nil
	}
	// Treat <br> as a line break.
	html = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(html, "\n")
	// Strip remaining tags.
	html = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, "")
	// Normalize whitespace: HTML entities and NBSP both collapse to ASCII space.
	html = strings.NewReplacer(
		"&nbsp;", " ", "\u00A0", " ",
		"&amp;", "&",
	).Replace(html)

	var out []string
	for _, line := range strings.Split(html, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// extractAddress finds "Street N" and "PLZ City" within a line list and joins them.
func extractAddress(lines []string) string {
	var street, plzCity string
	for _, ln := range lines {
		if street == "" {
			if m := streetRe.FindStringSubmatch(ln); len(m) == 2 {
				street = m[1]
			}
		}
		if plzCity == "" {
			if m := plzCityRe.FindStringSubmatch(ln); len(m) == 3 {
				plzCity = m[1] + " " + m[2]
			}
		}
	}
	switch {
	case street != "" && plzCity != "":
		return street + ", " + plzCity
	case plzCity != "":
		return plzCity
	case street != "":
		return street
	}
	return ""
}

// looksLikePersonName does a cheap heuristic: "Firstname Lastname" (optionally more words),
// no digits, not too long, no address keywords.
func looksLikePersonName(s string) bool {
	if s == "" || len(s) > 60 {
		return false
	}
	if strings.ContainsAny(s, "0123456789") {
		return false
	}
	lower := strings.ToLower(s)
	for _, kw := range []string{"strasse", "weg ", "gasse", "platz", "kita", "telefon", "email", "ostermundigen", "bern"} {
		if strings.Contains(lower, kw) {
			return false
		}
	}
	words := strings.Fields(s)
	return len(words) >= 2 && len(words) <= 5
}

// deobfuscateEmail turns "info(a)kita-laenggasse.ch" into "info@kita-laenggasse.ch".
func deobfuscateEmail(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "(a)", "@")
	s = strings.ReplaceAll(s, "[at]", "@")
	return s
}

// absoluteURL resolves a possibly-relative src against a base URL.
func absoluteURL(base, src string) string {
	if strings.HasPrefix(src, "http") {
		return src
	}
	if strings.HasPrefix(src, "//") {
		return "https:" + src
	}
	// Derive origin from base (first 3 slashes).
	const scheme = "https://"
	if !strings.HasPrefix(base, scheme) {
		return src
	}
	rest := base[len(scheme):]
	if i := strings.Index(rest, "/"); i >= 0 {
		rest = rest[:i]
	}
	origin := scheme + rest
	if strings.HasPrefix(src, "/") {
		return origin + src
	}
	return origin + "/" + src
}
