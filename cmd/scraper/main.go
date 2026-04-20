// scraper extracts Kita data from provider websites and writes a standard Excel file
// that can be imported into the Kita Springer Manager app.
//
// Usage:
//
//	go run ./cmd/scraper --source=stadt_bern --output=kitas_stadt_bern.xlsx
//	go run ./cmd/scraper --source=stiftung_bern --output=kitas_stiftung_bern.xlsx
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pak/kita-springer-manager/cmd/scraper/sources"
)

func main() {
	source := flag.String("source", "stadt_bern", "Data source to scrape (stadt_bern, stiftung_bern)")
	output := flag.String("output", "", "Output Excel file (default: <source>.xlsx)")
	flag.Parse()

	if *output == "" {
		*output = *source + ".xlsx"
	}

	log.Printf("Scraping source=%q → %s", *source, *output)

	var err error
	switch *source {
	case "stadt_bern":
		err = sources.ScrapeBern(*output)
	case "stiftung_bern":
		err = sources.ScrapeStiftung(*output)
	default:
		fmt.Fprintf(os.Stderr, "unknown source %q\nAvailable: stadt_bern, stiftung_bern\n", *source)
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("scrape error: %v", err)
	}
	log.Printf("Done — written to %s", *output)
}
