package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ChartConfig struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Slug        string `json:"slug,omitempty"`
	Direction   string `json:"direction"`
}

type DataPoint struct {
	Year  int     `json:"year"`
	Value float64 `json:"value"`
}

type ChartData struct {
	ChartConfig
	Year         int         `json:"year"`
	Value        *float64    `json:"value"`
	ValueColumn  string      `json:"valueColumn"`
	DataPoints   []DataPoint `json:"dataPoints"`
	LastUpdated  string      `json:"lastUpdated"`
}

var chartConfigs = []ChartConfig{
	// Bad Things Decreasing
	{ID: "child-mortality", Title: "Children Dying", Description: "The percentage of children dying before their fifth birthday has dramatically declined.", Source: "owid", Slug: "child-mortality", Direction: "down"},
	{ID: "hiv-infections", Title: "HIV Infections", Description: "The rate of new HIV infections per million people has been falling.", Source: "owid", Slug: "new-hiv-infections", Direction: "down"},
	{ID: "battle-deaths", Title: "Battle Deaths", Description: "The number of deaths in state-based conflicts per 100,000 people has fallen to historic lows.", Source: "owid", Slug: "state-based-battle-related-deaths-per-100000-since-1946", Direction: "down"},
	{ID: "oil-spills", Title: "Oil Spills", Description: "The volume of oil spilled from tankers has significantly decreased.", Source: "owid", Slug: "quantity-oil-spills", Direction: "down"},
	{ID: "solar-price", Title: "Expensive Solar Panels", Description: "The price of solar energy has plummeted.", Source: "owid", Slug: "solar-pv-prices", Direction: "down"},
	{ID: "so2-emissions", Title: "Smoke Particles", Description: "The amount of smoke particles (SO₂) emitted per person has decreased.", Source: "owid", Slug: "so-emissions-by-world-region-in-million-tonnes", Direction: "down"},
	{ID: "ozone-depletion", Title: "Ozone Depletion", Description: "The production of ozone-depleting substances has been almost entirely phased out.", Source: "owid", Slug: "ozone-depleting-substance-consumption", Direction: "down"},
	{ID: "plane-deaths", Title: "Plane Crash Deaths", Description: "The number of deaths per 10 billion passenger miles has drastically reduced.", Source: "owid", Slug: "global-fatalities-from-aviation-accidents-and-hijackings", Direction: "down"},
	{ID: "disaster-deaths", Title: "Deaths from Disaster", Description: "The number of people killed by natural disasters has fallen significantly over the last century.", Source: "owid", Slug: "number-of-deaths-from-natural-disasters", Direction: "down"},
	{ID: "nuclear-warheads", Title: "Nuclear Arms", Description: "The total number of nuclear warheads has been reduced since the Cold War peak.", Source: "owid", Slug: "nuclear-warhead-stockpiles", Direction: "down"},
	{ID: "child-labor", Title: "Child Labor", Description: "The percentage of children aged 5-14 who work full-time under bad conditions has decreased.", Source: "owid", Slug: "children-in-employment-total-percent-of-children-ages-7-14", Direction: "down"},
	{ID: "hunger", Title: "Hunger", Description: "The share of people who are undernourished has been falling.", Source: "owid", Slug: "prevalence-of-undernourishment", Direction: "down"},
	{ID: "extreme-poverty", Title: "Extreme Poverty", Description: "The share of humanity living on less than $3/day has fallen from >40% in 1981 to <10% today.", Source: "owid", Slug: "share-of-population-in-extreme-poverty", Direction: "down"},
	{ID: "maternal-mortality", Title: "Maternal Deaths", Description: "Global maternal deaths have more than halved since 1990.", Source: "owid", Slug: "maternal-mortality", Direction: "down"},
	{ID: "malaria-deaths", Title: "Malaria Deaths", Description: "Age-standardised malaria deaths per 100,000 people have dropped by ~45% since 2000.", Source: "owid", Slug: "malaria-death-rates", Direction: "down"},
	{ID: "co2-intensity", Title: "CO₂ Intensity", Description: "Each dollar of world GDP now emits ~40% less CO₂ than in 1990.", Source: "owid", Slug: "co2-intensity", Direction: "down"},

	// Good Things Increasing
	{ID: "protected-areas", Title: "Protected Nature", Description: "The share of the Earth's land surface that is protected as national parks and other reserves has increased.", Source: "owid", Slug: "terrestrial-protected-areas", Direction: "up"},
	{ID: "womens-suffrage", Title: "Women's Right to Vote", Description: "The share of countries where women have the right to vote has risen to include nearly all nations.", Source: "owid", Slug: "universal-suffrage-women-lexical", Direction: "up"},
	{ID: "cereal-yield", Title: "Harvest", Description: "The amount of cereal yield (in tonnes per hectare) has increased, meaning more food from the same land.", Source: "owid", Slug: "cereal-yield", Direction: "up"},
	{ID: "literacy", Title: "Literacy", Description: "The share of adults who are literate has risen dramatically.", Source: "owid", Slug: "literacy-rate-adults", Direction: "up"},
	{ID: "democracy", Title: "Democracy", Description: "The share of humanity living in a democracy has increased significantly.", Source: "owid", Slug: "people-living-in-democracies-autocracies", Direction: "up"},
	{ID: "girls-school", Title: "Girls in School", Description: "The share of girls of primary school age who are enrolled has risen to near-parity with boys.", Source: "owid", Slug: "net-enrollment-rate-primary-gender-parity-index-gpi", Direction: "up"},
	{ID: "electricity", Title: "Electricity Coverage", Description: "The share of people with some access to electricity has grown.", Source: "owid", Slug: "share-of-the-population-with-access-to-electricity", Direction: "up"},
	{ID: "mobile-phones", Title: "Mobile Phones", Description: "The share of people with a mobile phone subscription has skyrocketed.", Source: "owid", Slug: "mobile-cellular-subscriptions-per-100-people", Direction: "up"},
	{ID: "water-access", Title: "Water", Description: "The share of people with access to a protected water source has increased.", Source: "owid", Slug: "population-using-at-least-basic-drinking-water", Direction: "up"},
	{ID: "internet", Title: "Internet", Description: "The share of people using the internet has seen rapid growth.", Source: "owid", Slug: "share-of-individuals-using-the-internet", Direction: "up"},
	{ID: "immunization", Title: "Immunization", Description: "The share of 1-year-olds who have received at least one vaccination has greatly increased.", Source: "owid", Slug: "share-of-one-year-olds-vaccinated-against-dtp3", Direction: "up"},
	{ID: "scientific-papers", Title: "Science", Description: "The number of scholarly articles published per year has seen exponential growth.", Source: "owid", Slug: "scientific-and-technical-journal-articles", Direction: "up"},
	{ID: "renewable-energy", Title: "Clean Power", Description: "Renewables' share of global electricity has doubled in a decade, passing 30% in 2024.", Source: "owid", Slug: "share-of-electricity-production-from-renewable-sources", Direction: "up"},
	{ID: "life-expectancy", Title: "Life Expectancy", Description: "Average life expectancy has climbed from 52 years in 1960 to 73 years in 2023.", Source: "owid", Slug: "life-expectancy", Direction: "up"},

	// Manual placeholders
	{ID: "legal-slavery", Title: "Legal Slavery", Description: "The share of countries where slavery is legal has dropped to zero.", Source: "manual", Direction: "down"},
	{ID: "death-penalty", Title: "Death Penalty", Description: "The number of countries that have abolished the death penalty has steadily increased.", Source: "manual", Direction: "up"},
	{ID: "leaded-gasoline", Title: "Leaded Gasoline", Description: "The number of countries using leaded gasoline has dropped to just a few.", Source: "manual", Direction: "down"},
	{ID: "smallpox", Title: "Smallpox", Description: "The number of countries with smallpox has been eliminated (the disease is eradicated).", Source: "manual", Direction: "down"},
}

type Fetcher struct {
	client  *http.Client
	limiter *rate.Limiter
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: rate.NewLimiter(rate.Every(500*time.Millisecond), 1), // 2 requests per second
	}
}

func (f *Fetcher) fetchOWIDData(ctx context.Context, slug string) ([][]string, error) {
	// Rate limiting
	if err := f.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	url := fmt.Sprintf("https://ourworldindata.org/grapher/%s.csv", slug)
	log.Printf("Fetching: %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing CSV: %w", err)
	}

	return records, nil
}

func (f *Fetcher) processChartData(records [][]string, config ChartConfig) *ChartData {
	if len(records) < 2 {
		log.Printf("No data for %s", config.ID)
		return nil
	}

	headers := records[0]
	entityCol := findColumn(headers, []string{"Entity", "Country"})
	yearCol := findColumn(headers, []string{"Year"})
	
	if entityCol == -1 || yearCol == -1 {
		log.Printf("Missing required columns for %s", config.ID)
		return nil
	}

	// Find world data
	var worldRows [][]string
	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) > entityCol {
			entity := strings.ToLower(row[entityCol])
			if entity == "world" || entity == "global" {
				worldRows = append(worldRows, row)
			}
		}
	}

	if len(worldRows) == 0 {
		log.Printf("No world data found for %s", config.ID)
		return nil
	}

	// Find value column (prefer meaningful columns over metadata)
	valueCol := -1
	bestScore := -1
	
	for i, header := range headers {
		if i == yearCol || i == entityCol || strings.Contains(strings.ToLower(header), "code") {
			continue
		}
		
		// Score columns by relevance
		score := 0
		headerLower := strings.ToLower(header)
		
		// Prefer main data columns
		if strings.Contains(headerLower, "share") || 
		   strings.Contains(headerLower, "percent") ||
		   strings.Contains(headerLower, "rate") ||
		   strings.Contains(headerLower, "expectancy") ||
		   strings.Contains(headerLower, "literacy") {
			score += 10
		}
		
		// Chart-specific preferences
		if config.ID == "democracy" && strings.Contains(headerLower, "electoral democracies") {
			score += 20
		}
		if config.ID == "girls-school" && strings.Contains(headerLower, "parity") {
			score += 20
		}
		
		// Avoid metadata columns
		if strings.Contains(headerLower, "without") ||
		   strings.Contains(headerLower, "missing") ||
		   strings.Contains(headerLower, "unavailable") {
			score -= 20
		}
		
		// Shorter column names are often better
		if len(header) < 50 {
			score += 5
		}
		
		if score > bestScore {
			bestScore = score
			valueCol = i
		}
	}
	
	// Fallback to first numeric column if no good match
	if valueCol == -1 {
		for i, header := range headers {
			if i != yearCol && i != entityCol && !strings.Contains(strings.ToLower(header), "code") {
				valueCol = i
				break
			}
		}
	}

	if valueCol == -1 {
		log.Printf("No value column found for %s", config.ID)
		return nil
	}

	// Parse and sort data
	var dataPoints []DataPoint
	for _, row := range worldRows {
		if len(row) <= yearCol || len(row) <= valueCol {
			continue
		}

		year, err := strconv.Atoi(row[yearCol])
		if err != nil {
			continue
		}

		value, err := strconv.ParseFloat(row[valueCol], 64)
		if err != nil {
			continue
		}

		dataPoints = append(dataPoints, DataPoint{
			Year:  year,
			Value: value,
		})
	}

	if len(dataPoints) == 0 {
		log.Printf("No valid data points for %s", config.ID)
		return nil
	}

	// Sort by year (newest first)
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Year > dataPoints[j].Year
	})

	// Get latest value
	latest := dataPoints[0]
	latestValue := latest.Value

	// Include more historical data for wider date ranges
	// Sort by year (oldest first for better visualization)
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Year < dataPoints[j].Year
	})
	
	// Take every N-th point to get good coverage across time range
	if len(dataPoints) > 20 {
		step := len(dataPoints) / 20
		if step < 1 {
			step = 1
		}
		var sampledPoints []DataPoint
		for i := 0; i < len(dataPoints); i += step {
			sampledPoints = append(sampledPoints, dataPoints[i])
		}
		// Always include the most recent point
		if len(dataPoints) > 0 {
			latest := dataPoints[len(dataPoints)-1]
			if len(sampledPoints) == 0 || sampledPoints[len(sampledPoints)-1].Year != latest.Year {
				sampledPoints = append(sampledPoints, latest)
			}
		}
		dataPoints = sampledPoints
	}

	return &ChartData{
		ChartConfig: config,
		Year:        latest.Year,
		Value:       &latestValue,
		ValueColumn: headers[valueCol],
		DataPoints:  dataPoints,
		LastUpdated: time.Now().Format(time.RFC3339),
	}
}

func findColumn(headers []string, candidates []string) int {
	for i, header := range headers {
		for _, candidate := range candidates {
			if strings.EqualFold(header, candidate) {
				return i
			}
		}
	}
	return -1
}

func main() {
	ctx := context.Background()
	fetcher := NewFetcher()

	// Create output directory
	dataDir := filepath.Join("data", "charts")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Creating data directory: %v", err)
	}

	results := make(map[string]*ChartData)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process OWID charts concurrently
	for _, config := range chartConfigs {
		if config.Source != "owid" {
			continue
		}

		wg.Add(1)
		go func(c ChartConfig) {
			defer wg.Done()

			records, err := fetcher.fetchOWIDData(ctx, c.Slug)
			if err != nil {
				log.Printf("Error fetching %s: %v", c.ID, err)
				return
			}

			chartData := fetcher.processChartData(records, c)
			if chartData != nil {
				mu.Lock()
				results[c.ID] = chartData
				mu.Unlock()
				log.Printf("✓ Processed %s", c.Title)
			}
		}(config)
	}

	wg.Wait()

	// Add manual data placeholders
	for _, config := range chartConfigs {
		if config.Source == "manual" {
			results[config.ID] = &ChartData{
				ChartConfig: config,
				Year:        2024,
				Value:       nil,
				ValueColumn: "Manual Data Required",
				DataPoints:  []DataPoint{},
				LastUpdated: time.Now().Format(time.RFC3339),
			}
		}
	}

	// Save results
	outputFile := filepath.Join(dataDir, "chart-data.json")
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("Marshaling JSON: %v", err)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		log.Fatalf("Writing output file: %v", err)
	}

	log.Printf("✓ Data saved to %s", outputFile)
	log.Printf("✓ Successfully processed %d charts", len(results))
}