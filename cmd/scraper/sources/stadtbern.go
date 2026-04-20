// Package sources implements web scrapers for Kita data providers.
// Each scraper fetches structured Kita data and writes it as a standard Excel file.
//
// Standard Excel columns (row 1 = header, row 2+ = data):
//
//	A: Name          B: Adresse              C: ÖV-Haltestelle
//	D: Telefon       E: Email                F: Gruppen (semicolon-separated)
//	G: Notizen
package sources

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/xuri/excelize/v2"
)

const (
	bernBaseURL  = "https://www.bern.ch/themen/kinder-jugendliche-und-familie/kinderbetreuung/kitas-stadt-bern/angebot/unsere-kitas"
	bernProvider = "Kitas Stadt Bern"
)

type kitaData struct {
	Name        string
	Address     string
	StopName    string
	Phone       string
	Email       string
	Groups      []string
	Notes       string
	LeitungName string
	PhotoURL    string
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

func fetch(url string) (*goquery.Document, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	return goquery.NewDocumentFromReader(resp.Body)
}

// ScrapeBern fetches all Kitas from Kitas Stadt Bern and writes the standard Excel.
func ScrapeBern(outputPath string) error {
	slugs, err := bernKitaSlugs()
	if err != nil {
		return fmt.Errorf("fetch kita list: %w", err)
	}
	log.Printf("Found %d Kitas", len(slugs))

	var kitas []kitaData
	for _, slug := range slugs {
		k, err := bernKitaDetail(slug)
		if err != nil {
			log.Printf("  WARN: %s: %v", slug, err)
			kitas = append(kitas, kitaData{Name: bernSlugToName(slug)})
			continue
		}
		log.Printf("  OK: %s — %d groups", k.Name, len(k.Groups))
		kitas = append(kitas, *k)
	}

	return writeExcel(outputPath, bernProvider, kitas)
}

// bernKitaSlugs fetches the overview page and returns all Kita URL slugs.
func bernKitaSlugs() ([]string, error) {
	doc, err := fetch(bernBaseURL)
	if err != nil {
		return nil, err
	}

	var slugs []string
	seen := map[string]bool{}

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		// Match links like ".../unsere-kitas/kita-xxxxx" but not sub-pages
		if !strings.Contains(href, "/unsere-kitas/kita-") {
			return
		}
		// Extract the kita slug (last path segment)
		parts := strings.Split(strings.TrimSuffix(href, "/"), "/")
		slug := parts[len(parts)-1]
		if strings.HasPrefix(slug, "kita-") && !seen[slug] {
			seen[slug] = true
			slugs = append(slugs, slug)
		}
	})
	return slugs, nil
}

// bernKitaDetail fetches address, phone, email and groups for one Kita.
func bernKitaDetail(slug string) (*kitaData, error) {
	kitaURL := bernBaseURL + "/" + slug

	// Fetch main page for basic info
	doc, err := fetch(kitaURL)
	if err != nil {
		return nil, err
	}

	k := &kitaData{
		Name: bernSlugToName(slug),
	}

	// Try to extract name from page title or heading
	if h := doc.Find("h1").First().Text(); h != "" {
		k.Name = strings.TrimSpace(h)
	}

	// Phone: look for tel: links
	doc.Find("a[href^='tel:']").Each(func(_ int, s *goquery.Selection) {
		if k.Phone == "" {
			href, _ := s.Attr("href")
			k.Phone = strings.TrimPrefix(href, "tel:")
			// Also try text content as it's usually formatted nicely
			if txt := strings.TrimSpace(s.Text()); txt != "" {
				k.Phone = txt
			}
		}
	})

	// Email: look for mailto: links
	doc.Find("a[href^='mailto:']").Each(func(_ int, s *goquery.Selection) {
		if k.Email == "" {
			href, _ := s.Attr("href")
			k.Email = strings.TrimPrefix(href, "mailto:")
		}
	})

	// Address: look in structured address blocks or common address containers
	addressCandidates := []string{
		".address", ".adr", "[itemprop='address']",
		".contact-address", ".location",
		"address",
	}
	for _, sel := range addressCandidates {
		if addr := strings.TrimSpace(doc.Find(sel).First().Text()); addr != "" {
			k.Address = normalizeWhitespace(addr)
			break
		}
	}

	// Fetch address sub-page if address still empty
	if k.Address == "" {
		if addr, phone, email := bernAddressBlock(slug); addr != "" {
			k.Address = addr
			if k.Phone == "" {
				k.Phone = phone
			}
			if k.Email == "" {
				k.Email = email
			}
		}
	}

	// Photo: first content image on page (skip icons/logos by min-size heuristic via alt or src)
	doc.Find("img[src]").Each(func(_ int, s *goquery.Selection) {
		if k.PhotoURL != "" {
			return
		}
		src, _ := s.Attr("src")
		if src == "" || strings.Contains(src, "logo") || strings.Contains(src, "icon") || strings.Contains(src, "sprite") {
			return
		}
		if strings.HasPrefix(src, "http") {
			k.PhotoURL = src
		} else if strings.HasPrefix(src, "/") {
			k.PhotoURL = "https://www.bern.ch" + src
		}
	})

	// Leitung: look for text directly following a "Leitung" label
	doc.Find("dt, th, strong, b, label").Each(func(_ int, s *goquery.Selection) {
		if k.LeitungName != "" {
			return
		}
		txt := strings.TrimSpace(s.Text())
		if strings.EqualFold(txt, "leitung") || strings.EqualFold(txt, "kita-leitung") || strings.EqualFold(txt, "leiter/in") {
			// Try sibling dd/td first
			if sibling := s.Next(); sibling != nil {
				if name := strings.TrimSpace(sibling.Text()); name != "" && len(name) < 80 {
					k.LeitungName = name
					return
				}
			}
			// Try parent's next sibling
			if sibling := s.Parent().Next(); sibling != nil {
				if name := strings.TrimSpace(sibling.Text()); name != "" && len(name) < 80 {
					k.LeitungName = name
				}
			}
		}
	})

	// Fetch groups sub-page
	k.Groups = bernGroups(slug)

	return k, nil
}

// bernAddressBlock fetches the dedicated address sub-page for a Kita.
func bernAddressBlock(slug string) (address, phone, email string) {
	url := bernBaseURL + "/" + slug + "/bern-web-addressblock/addressblock_detail_view"
	doc, err := fetch(url)
	if err != nil {
		return
	}

	// Extract all text and look for address patterns
	doc.Find("p, div, span, li").Each(func(_ int, s *goquery.Selection) {
		txt := strings.TrimSpace(s.Text())
		if len(txt) > 5 && len(txt) < 200 {
			lower := strings.ToLower(txt)
			if address == "" && (strings.Contains(lower, "strasse") ||
				strings.Contains(lower, "weg") ||
				strings.Contains(lower, "gasse") ||
				strings.Contains(lower, "platz")) {
				address = normalizeWhitespace(txt)
			}
		}
	})

	doc.Find("a[href^='tel:']").Each(func(_ int, s *goquery.Selection) {
		if phone == "" {
			phone = strings.TrimSpace(s.Text())
		}
	})
	doc.Find("a[href^='mailto:']").Each(func(_ int, s *goquery.Selection) {
		if email == "" {
			href, _ := s.Attr("href")
			email = strings.TrimPrefix(href, "mailto:")
		}
	})
	return
}

// bernGroups fetches the groups sub-page for a Kita.
func bernGroups(slug string) []string {
	url := bernBaseURL + "/" + slug + "/gruppen"
	doc, err := fetch(url)
	if err != nil {
		return nil
	}

	var groups []string
	seen := map[string]bool{}

	// Group names are typically in headings on the groups page
	doc.Find("h2, h3, h4, .group-name, .gruppe").Each(func(_ int, s *goquery.Selection) {
		name := strings.TrimSpace(s.Text())
		if name == "" || seen[name] || len(name) > 50 {
			return
		}
		// Skip navigation/page headings
		lower := strings.ToLower(name)
		skip := []string{"gruppen", "navigation", "inhalt", "kontakt", "öffnungszeiten", "anmeldung", "angebot"}
		for _, kw := range skip {
			if strings.Contains(lower, kw) {
				return
			}
		}
		seen[name] = true
		groups = append(groups, name)
	})
	return groups
}

// bernSlugToName converts "kita-ausserholligen" to "Kita Ausserholligen".
func bernSlugToName(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func normalizeWhitespace(s string) string {
	lines := strings.Fields(s)
	return strings.Join(lines, " ")
}

// writeExcel writes Kita data to the standard import Excel format.
func writeExcel(path, providerName string, kitas []kitaData) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Kitas"
	f.SetSheetName("Sheet1", sheet)

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#2563EB"}, Pattern: 1},
	})

	headers := []string{"Name", "Adresse", "ÖV-Haltestelle", "Telefon", "Email", "Gruppen", "Notizen", "Leitung", "Foto-URL"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Provider info in a comment cell
	f.SetCellValue(sheet, "A1", fmt.Sprintf("Name [Träger: %s]", providerName))

	for row, k := range kitas {
		r := row + 2
		f.SetCellValue(sheet, cellName(1, r), k.Name)
		f.SetCellValue(sheet, cellName(2, r), k.Address)
		f.SetCellValue(sheet, cellName(3, r), k.StopName)
		f.SetCellValue(sheet, cellName(4, r), k.Phone)
		f.SetCellValue(sheet, cellName(5, r), k.Email)
		f.SetCellValue(sheet, cellName(6, r), strings.Join(k.Groups, "; "))
		f.SetCellValue(sheet, cellName(7, r), k.Notes)
		f.SetCellValue(sheet, cellName(8, r), k.LeitungName)
		f.SetCellValue(sheet, cellName(9, r), k.PhotoURL)
	}

	// Column widths
	widths := []float64{28, 38, 28, 16, 30, 35, 25, 28, 45}
	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	for i, col := range cols {
		f.SetColWidth(sheet, col, col, widths[i])
	}

	return f.SaveAs(path)
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
