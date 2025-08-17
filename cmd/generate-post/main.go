package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DataPoint struct {
	Year  int     `json:"year"`
	Value float64 `json:"value"`
}

type ChartData struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Source      string      `json:"source"`
	Slug        string      `json:"slug,omitempty"`
	Direction   string      `json:"direction"`
	Year        int         `json:"year"`
	Value       *float64    `json:"value"`
	ValueColumn string      `json:"valueColumn"`
	DataPoints  []DataPoint `json:"dataPoints"`
	LastUpdated string      `json:"lastUpdated"`
}

func formatValue(value *float64, column string) string {
	if value == nil {
		return "Data not available"
	}

	v := *value
	columnLower := strings.ToLower(column)

	// Detect percentage fields more broadly
	if strings.Contains(columnLower, "percent") || 
	   strings.Contains(columnLower, "share") ||
	   strings.Contains(columnLower, "rate") ||
	   strings.Contains(columnLower, "coverage") ||
	   strings.Contains(columnLower, "access") ||
	   (v <= 100 && (strings.Contains(columnLower, "population") || strings.Contains(columnLower, "people"))) {
		return fmt.Sprintf("%.1f%%", v)
	}
	
	// Years
	if strings.Contains(columnLower, "year") || strings.Contains(columnLower, "expectancy") {
		return fmt.Sprintf("%.1f years", v)
	}
	
	// Large numbers
	if v > 1000000 {
		return fmt.Sprintf("%.1fM", v/1000000)
	} else if v > 1000 {
		return fmt.Sprintf("%.1fK", v/1000)
	} else if v < 1 && v > 0 {
		return fmt.Sprintf("%.3f", v)
	} else {
		return fmt.Sprintf("%.1f", v)
	}
}

func generatePost(chartDataMap map[string]*ChartData) string {
	lastUpdated := time.Now().Format("2006-01-02")

	// Separate charts by direction
	var badThings, goodThings []*ChartData
	for _, chart := range chartDataMap {
		if chart.Direction == "down" {
			badThings = append(badThings, chart)
		} else {
			goodThings = append(goodThings, chart)
		}
	}

	// Build the post
	var post strings.Builder

	// Front matter
	post.WriteString(fmt.Sprintf(`---
title: "32 Optimistic Charts: The World is Getting Better"
date: %s
tags: ["progress", "data", "optimism"]
description: "Data-driven evidence that humanity is making remarkable progress across health, education, technology, and human rights."
draft: false
---

The world often feels like it's falling apart, but the data tells a different story. Here are 32 charts showing measurable human progress across health, education, technology, environment, and human rights.

*Last updated: %s*

## 16 Bad Things That Are Decreasing
`, lastUpdated, lastUpdated))

	// Bad things section
	for _, chart := range badThings {
		latestValue := ""
		if chart.Value != nil {
			latestValue = fmt.Sprintf(" Latest data (%d): %s", chart.Year, formatValue(chart.Value, chart.ValueColumn))
		}

		post.WriteString(fmt.Sprintf(`
### %s

%s%s

{{< chart id="%s" >}}
`, chart.Title, chart.Description, latestValue, chart.ID))
	}

	// Good things section
	post.WriteString(`
## 16 Good Things That Are Increasing

`)

	for _, chart := range goodThings {
		latestValue := ""
		if chart.Value != nil {
			latestValue = fmt.Sprintf(" Latest data (%d): %s", chart.Year, formatValue(chart.Value, chart.ValueColumn))
		}

		post.WriteString(fmt.Sprintf(`
### %s

%s%s

{{< chart id="%s" >}}
`, chart.Title, chart.Description, latestValue, chart.ID))
	}

	// Conclusion
	post.WriteString(`
## The Big Picture

These charts represent decades of human effort, innovation, and cooperation. While challenges remain, the trajectory is clear: by most measures that matter for human wellbeing, we are making remarkable progress.

The data comes primarily from [Our World in Data](https://ourworldindata.org/), with additional sources including Gapminder, WHO, IMDb, and Discogs.

---

*This post is automatically updated monthly with the latest available data.*
`)

	return post.String()
}

func main() {
	// Read chart data
	dataFile := filepath.Join("data", "charts", "chart-data.json")
	data, err := os.ReadFile(dataFile)
	if err != nil {
		log.Fatalf("Reading chart data: %v", err)
	}

	var chartDataMap map[string]*ChartData
	if err := json.Unmarshal(data, &chartDataMap); err != nil {
		log.Fatalf("Parsing chart data: %v", err)
	}

	// Generate Hugo post
	post := generatePost(chartDataMap)

	// Write to content directory
	outputDir := filepath.Join("content", "writing")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Creating output directory: %v", err)
	}

	outputFile := filepath.Join(outputDir, "32-optimistic-charts.md")
	if err := os.WriteFile(outputFile, []byte(post), 0644); err != nil {
		log.Fatalf("Writing post file: %v", err)
	}

	log.Printf("✓ Generated Hugo post: %s", outputFile)

	// Also generate data file for Hugo shortcodes
	shortcodeDataDir := "data"
	if err := os.MkdirAll(shortcodeDataDir, 0755); err != nil {
		log.Fatalf("Creating shortcode data directory: %v", err)
	}

	shortcodeDataFile := filepath.Join(shortcodeDataDir, "optimistic_charts.json")
	formattedData, err := json.MarshalIndent(chartDataMap, "", "  ")
	if err != nil {
		log.Fatalf("Marshaling shortcode data: %v", err)
	}

	if err := os.WriteFile(shortcodeDataFile, formattedData, 0644); err != nil {
		log.Fatalf("Writing shortcode data: %v", err)
	}

	log.Printf("✓ Generated data file for Hugo shortcodes: %s", shortcodeDataFile)
	log.Printf("✓ Successfully processed %d charts", len(chartDataMap))
}