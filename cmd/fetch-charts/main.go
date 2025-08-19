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
	{ID: "plane-deaths", Title: "Plane Crash Deaths", Description: "Aviation fatalities per million passengers have dramatically declined, making flying safer than ever.", Source: "owid", Slug: "aviation-fatalities-per-million-passengers", Direction: "down"},
	{ID: "disaster-deaths", Title: "Deaths from Disaster", Description: "The number of people killed by natural disasters has fallen significantly over the last century.", Source: "owid", Slug: "number-of-deaths-from-natural-disasters", Direction: "down"},
	{ID: "nuclear-warheads", Title: "Nuclear Arms", Description: "The total number of nuclear warheads has been reduced since the Cold War peak.", Source: "owid", Slug: "nuclear-warhead-stockpiles", Direction: "down"},
	{ID: "child-labor", Title: "Child Labor", Description: "The percentage of children aged 5-14 who work full-time under bad conditions has decreased.", Source: "owid", Slug: "children-in-employment-total-percent-of-children-ages-7-14", Direction: "down"},
	{ID: "hunger", Title: "Hunger", Description: "The share of people who are undernourished has been falling.", Source: "owid", Slug: "prevalence-of-undernourishment", Direction: "down"},
	{ID: "extreme-poverty", Title: "Extreme Poverty", Description: "The share of humanity living on less than $3/day has fallen from >40% in 1981 to <10% today.", Source: "owid", Slug: "share-of-population-in-extreme-poverty", Direction: "down"},
	{ID: "maternal-mortality", Title: "Maternal Deaths", Description: "Global maternal deaths have more than halved since 1990.", Source: "owid", Slug: "maternal-mortality", Direction: "down"},
	{ID: "malaria-deaths", Title: "Malaria Deaths", Description: "Age-standardised malaria deaths per 100,000 people have dropped by ~45% since 2000.", Source: "owid", Slug: "malaria-death-rates", Direction: "down"},
	{ID: "co2-intensity", Title: "CO₂ Intensity", Description: "Each dollar of world GDP now emits ~40% less CO₂ than in 1990.", Source: "owid", Slug: "co2-intensity", Direction: "down"},

	// Good Things Increasing

	{ID: "womens-suffrage", Title: "Women's Right to Vote", Description: "The number of countries where women have the right to vote has grown from 1 in 1893 to 195 today.", Source: "manual", Direction: "up"},
	{ID: "cereal-yield", Title: "Harvest", Description: "The amount of cereal yield (in tonnes per hectare) has increased, meaning more food from the same land.", Source: "owid", Slug: "cereal-yield", Direction: "up"},
	{ID: "literacy", Title: "Literacy", Description: "The share of adults who are literate has risen dramatically.", Source: "owid", Slug: "literacy-rate-adults", Direction: "up"},
	{ID: "democracy", Title: "Democracy", Description: "The number of countries that are electoral or liberal democracies has increased significantly.", Source: "owid", Slug: "countries-democracies-autocracies-row.csv?v=1&csvType=full&useColumnShortNames=true", Direction: "up"},
	{ID: "girls-school", Title: "Girls in School", Description: "The share of girls of primary school age who are enrolled has risen to near-parity with boys.", Source: "owid", Slug: "net-enrollment-rate-primary-gender-parity-index-gpi", Direction: "up"},
	{ID: "electricity", Title: "Electricity Coverage", Description: "The share of people with some access to electricity has grown.", Source: "owid", Slug: "share-of-the-population-with-access-to-electricity", Direction: "up"},
	{ID: "mobile-phones", Title: "Mobile Phones", Description: "The share of people with a mobile phone subscription has skyrocketed.", Source: "owid", Slug: "mobile-cellular-subscriptions-per-100-people", Direction: "up"},
	{ID: "water-access", Title: "Water", Description: "The share of people with access to a protected water source has increased.", Source: "owid", Slug: "population-using-at-least-basic-drinking-water", Direction: "up"},
	{ID: "internet", Title: "Internet", Description: "The share of people using the internet has seen rapid growth.", Source: "owid", Slug: "share-of-individuals-using-the-internet", Direction: "up"},
	{ID: "immunization", Title: "Immunization", Description: "The share of 1-year-olds who have received at least one vaccination has greatly increased.", Source: "owid", Slug: "share-of-one-year-olds-vaccinated-against-dtp3", Direction: "up"},
	{ID: "scientific-papers", Title: "Science", Description: "The number of scholarly articles published per year has seen exponential growth.", Source: "owid", Slug: "scientific-and-technical-journal-articles", Direction: "up"},
	{ID: "renewable-energy", Title: "Clean Power", Description: "Renewables' share of global electricity has doubled in a decade, passing 30% in 2024.", Source: "owid", Slug: "share-of-electricity-production-from-renewable-sources", Direction: "up"},
	{ID: "life-expectancy", Title: "Life Expectancy", Description: "Average life expectancy has climbed from 52 years in 1960 to 73 years in 2023.", Source: "owid", Slug: "life-expectancy", Direction: "up"},

	// More Bad Things Decreasing
	{ID: "smoking-prevalence", Title: "Smoking", Description: "Global smoking rates have fallen by roughly a quarter since 1990, saving millions of lives.", Source: "owid", Slug: "share-of-adults-who-smoke", Direction: "down"},
	{ID: "homicide-rate", Title: "Homicide Rate", Description: "Intentional killings per 100k people have fallen by a third since the early 1990s.", Source: "owid", Slug: "homicide-rate-unodc", Direction: "down"},



	// More Good Things Increasing
	{ID: "sanitation-access", Title: "Improved Sanitation", Description: "Two-thirds of humanity had safe sanitation in 2022, up from half in 2000.", Source: "owid", Slug: "share-using-safely-managed-sanitation", Direction: "up"},
	{ID: "protected-land", Title: "Protected Land", Description: "The share of land under legal protection has grown from virtually zero to nearly 18%.", Source: "manual", Direction: "up"},
	{ID: "monitored-species", Title: "Monitored Species", Description: "The number of species evaluated for conservation status has grown from 34 in 1959 to over 157,000 today.", Source: "manual", Direction: "up"},
	{ID: "comprehensive-vaccination", Title: "Comprehensive Vaccination", Description: "Share of one-year-olds with all six basic vaccines", Source: "manual", Direction: "up"},
	{ID: "child-cancer-survival", Title: "Child Cancer Survival", Description: "5-year survival rates for childhood cancer have improved from 58% in 1975 to 80% in 2010.", Source: "manual", Direction: "up"},


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

	base := "https://ourworldindata.org/grapher/"
	url := ""
	// Allow passing a full or partial path with query string or explicit .csv
	if strings.HasPrefix(slug, "http://") || strings.HasPrefix(slug, "https://") {
		url = slug
	} else if strings.Contains(slug, ".csv") || strings.Contains(slug, "?") {
		url = base + slug
	} else {
		url = fmt.Sprintf("%s%s.csv", base, slug)
	}
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
		if config.ID == "democracy" && (strings.Contains(headerLower, "electoral democracies") || strings.Contains(headerLower, "liberal democracies")) {
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
	
	// Special handling for democracy - combine electoral and liberal democracies
	if config.ID == "democracy" {
		dataPoints = f.processDemocracyData(headers, worldRows, yearCol)
	} else {
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

    valueColName := headers[valueCol]
    if config.ID == "democracy" {
        valueColName = "Electoral + Liberal democracies (countries)"
    }
    return &ChartData{
        ChartConfig: config,
        Year:        latest.Year,
        Value:       &latestValue,
        ValueColumn: valueColName,
        DataPoints:  dataPoints,
        LastUpdated: time.Now().Format(time.RFC3339),
    }
}

func (f *Fetcher) processDemocracyData(headers []string, worldRows [][]string, yearCol int) []DataPoint {
	// Find electoral and liberal democracy columns
	electoralCol := -1
	liberalCol := -1
	
    for i, header := range headers {
        headerLower := strings.ToLower(header)
        // Match both singular/plural and underscore/space variants
        if strings.Contains(headerLower, "electoral democracies") ||
           strings.Contains(headerLower, "electoral democracy") ||
           strings.Contains(headerLower, "electoral_democracy") {
            electoralCol = i
        }
        if strings.Contains(headerLower, "liberal democracies") ||
           strings.Contains(headerLower, "liberal democracy") ||
           strings.Contains(headerLower, "liberal_democracy") {
            liberalCol = i
        }
    }
	
	var dataPoints []DataPoint
	yearData := make(map[int]float64)
	
	// Process each row and sum electoral + liberal democracy values
	for _, row := range worldRows {
		if len(row) <= yearCol {
			continue
		}

		year, err := strconv.Atoi(row[yearCol])
		if err != nil {
			continue
		}

		totalValue := 0.0
		hasData := false
		
		// Add electoral democracies
		if electoralCol != -1 && len(row) > electoralCol {
			if val, err := strconv.ParseFloat(row[electoralCol], 64); err == nil {
				totalValue += val
				hasData = true
			}
		}
		
		// Add liberal democracies  
		if liberalCol != -1 && len(row) > liberalCol {
			if val, err := strconv.ParseFloat(row[liberalCol], 64); err == nil {
				totalValue += val
				hasData = true
			}
		}
		
		if hasData {
			yearData[year] = totalValue
		}
	}
	
	// Convert map to sorted slice
	for year, value := range yearData {
		dataPoints = append(dataPoints, DataPoint{
			Year:  year,
			Value: value,
		})
	}
	
	// Sort by year
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Year < dataPoints[j].Year
	})
	
	return dataPoints
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
			if config.ID == "legal-slavery" {
				// Specific data for legal slavery
				legalSlaveryData := []DataPoint{
					{Year: 1800, Value: 194},
					{Year: 1820, Value: 189},
					{Year: 1840, Value: 174},
					{Year: 1860, Value: 165},
					{Year: 1880, Value: 160},
					{Year: 1900, Value: 158},
					{Year: 1920, Value: 157},
					{Year: 1940, Value: 145},
					{Year: 1960, Value: 102},
					{Year: 1980, Value: 54},
					{Year: 2000, Value: 15},
					{Year: 2017, Value: 3},
				}
				results[config.ID] = &ChartData{
					ChartConfig: ChartConfig{
						ID:          config.ID,
						Title:       config.Title,
						Description: "The number of countries where slavery is legal has dropped from 194 in 1800 to just 3 today.",
						Source:      config.Source,
						Direction:   config.Direction,
					},
					Year:        2017,
					Value:       &[]float64{3}[0],
					ValueColumn: "Number of countries where slavery is legal",
					DataPoints:  legalSlaveryData,
					LastUpdated: time.Now().Format(time.RFC3339),
				}
			} else if config.ID == "womens-suffrage" {
				// Specific data for women's suffrage
				womensSuffrageData := []DataPoint{
					{Year: 1893, Value: 1},
					{Year: 1906, Value: 2},
					{Year: 1913, Value: 3},
					{Year: 1915, Value: 5},
					{Year: 1917, Value: 11},
					{Year: 1918, Value: 17},
					{Year: 1919, Value: 22},
					{Year: 1920, Value: 26},
					{Year: 1921, Value: 27},
					{Year: 1922, Value: 29},
					{Year: 1924, Value: 33},
					{Year: 1925, Value: 34},
					{Year: 1928, Value: 35},
					{Year: 1931, Value: 37},
					{Year: 1932, Value: 40},
					{Year: 1934, Value: 42},
					{Year: 1937, Value: 43},
					{Year: 1938, Value: 44},
					{Year: 1940, Value: 45},
					{Year: 1942, Value: 46},
					{Year: 1944, Value: 48},
					{Year: 1945, Value: 60},
					{Year: 1946, Value: 68},
					{Year: 1947, Value: 75},
					{Year: 1948, Value: 81},
					{Year: 1949, Value: 84},
					{Year: 1950, Value: 86},
					{Year: 1951, Value: 94},
					{Year: 1952, Value: 97},
					{Year: 1953, Value: 100},
					{Year: 1954, Value: 103},
					{Year: 1955, Value: 109},
					{Year: 1956, Value: 116},
					{Year: 1957, Value: 119},
					{Year: 1958, Value: 124},
					{Year: 1959, Value: 128},
					{Year: 1960, Value: 132},
					{Year: 1961, Value: 139},
					{Year: 1962, Value: 144},
					{Year: 1963, Value: 151},
					{Year: 1964, Value: 154},
					{Year: 1965, Value: 157},
					{Year: 1967, Value: 161},
					{Year: 1968, Value: 163},
					{Year: 1970, Value: 166},
					{Year: 1971, Value: 167},
					{Year: 1974, Value: 169},
					{Year: 1975, Value: 174},
					{Year: 1976, Value: 176},
					{Year: 1977, Value: 177},
					{Year: 1979, Value: 180},
					{Year: 1980, Value: 181},
					{Year: 1984, Value: 182},
					{Year: 1986, Value: 183},
					{Year: 1989, Value: 184},
					{Year: 1990, Value: 185},
					{Year: 1991, Value: 186},
					{Year: 1994, Value: 187},
					{Year: 1996, Value: 188},
					{Year: 1997, Value: 189},
					{Year: 2002, Value: 190},
					{Year: 2003, Value: 191},
					{Year: 2005, Value: 192},
					{Year: 2006, Value: 193},
					{Year: 2015, Value: 194},
					{Year: 2020, Value: 194},
					{Year: 2023, Value: 195},
				}
				results[config.ID] = &ChartData{
					ChartConfig: ChartConfig{
						ID:          config.ID,
						Title:       config.Title,
						Description: "The number of countries where women have the right to vote has grown from 1 in 1893 to 195 today.",
						Source:      config.Source,
						Direction:   config.Direction,
					},
					Year:        2023,
					Value:       &[]float64{195}[0],
					ValueColumn: "Number of countries",
					DataPoints:  womensSuffrageData,
					LastUpdated: time.Now().Format(time.RFC3339),
				}
			} else if config.ID == "monitored-species" {
				// Specific data for monitored species
				monitoredSpeciesData := []DataPoint{
					{Year: 1959, Value: 34},
					{Year: 1960, Value: 34},
					{Year: 1970, Value: 50},
					{Year: 1980, Value: 200},
					{Year: 1990, Value: 2500},
					{Year: 1995, Value: 5000},
					{Year: 2000, Value: 22456},
					{Year: 2002, Value: 25000},
					{Year: 2004, Value: 45000},
					{Year: 2006, Value: 45000},
					{Year: 2010, Value: 65000},
					{Year: 2012, Value: 75000},
					{Year: 2014, Value: 90000},
					{Year: 2016, Value: 105000},
					{Year: 2018, Value: 125000},
					{Year: 2020, Value: 135000},
					{Year: 2022, Value: 150000},
					{Year: 2023, Value: 157190},
				}
				results[config.ID] = &ChartData{
					ChartConfig: ChartConfig{
						ID:          config.ID,
						Title:       config.Title,
						Description: "The number of species evaluated for conservation status has grown from 34 in 1959 to over 157,000 today.",
						Source:      config.Source,
						Direction:   config.Direction,
					},
					Year:        2023,
					Value:       &[]float64{157190}[0],
					ValueColumn: "Evaluated Species",
					DataPoints:  monitoredSpeciesData,
					LastUpdated: time.Now().Format(time.RFC3339),
				}
			} else if config.ID == "comprehensive-vaccination" {
				// Specific data for comprehensive vaccination coverage
				vaccinationData := []DataPoint{
					{Year: 2000, Value: 13},
					{Year: 2001, Value: 14},
					{Year: 2002, Value: 17},
					{Year: 2003, Value: 18},
					{Year: 2004, Value: 19},
					{Year: 2005, Value: 20},
					{Year: 2006, Value: 21},
					{Year: 2007, Value: 25},
					{Year: 2008, Value: 28},
					{Year: 2009, Value: 38},
					{Year: 2010, Value: 40},
					{Year: 2011, Value: 43},
					{Year: 2012, Value: 45},
					{Year: 2013, Value: 51},
					{Year: 2014, Value: 55},
					{Year: 2015, Value: 63},
					{Year: 2016, Value: 70},
					{Year: 2017, Value: 71},
					{Year: 2018, Value: 73},
					{Year: 2019, Value: 74},
					{Year: 2020, Value: 73},
					{Year: 2021, Value: 72},
					{Year: 2022, Value: 76},
				}
				results[config.ID] = &ChartData{
					ChartConfig: ChartConfig{
						ID:          config.ID,
						Title:       config.Title,
						Description: "Share of one-year-olds with all six basic vaccines",
						Source:      config.Source,
						Direction:   config.Direction,
					},
					Year:        2022,
					Value:       &[]float64{76}[0],
					ValueColumn: "Lowest vaccination rate (%)",
					DataPoints:  vaccinationData,
					LastUpdated: time.Now().Format(time.RFC3339),
				}
			} else if config.ID == "child-cancer-survival" {
				// Specific data for child cancer survival
				childCancerData := []DataPoint{
					{Year: 1975, Value: 58},
					{Year: 1980, Value: 60},
					{Year: 1985, Value: 63},
					{Year: 1990, Value: 66},
					{Year: 1995, Value: 70},
					{Year: 2000, Value: 75},
					{Year: 2005, Value: 78},
					{Year: 2010, Value: 80},
				}
				results[config.ID] = &ChartData{
					ChartConfig: ChartConfig{
						ID:          config.ID,
						Title:       config.Title,
						Description: "5-year survival rates for childhood cancer have improved from 58% in 1975 to 80% in 2010.",
						Source:      config.Source,
						Direction:   config.Direction,
					},
					Year:        2010,
					Value:       &[]float64{80}[0],
					ValueColumn: "5-Year Survival (%)",
					DataPoints:  childCancerData,
					LastUpdated: time.Now().Format(time.RFC3339),
				}
			} else if config.ID == "protected-land" {
				// Specific data for protected land
				protectedLandData := []DataPoint{
					{Year: 1900, Value: 0.03},
					{Year: 1910, Value: 0.05},
					{Year: 1920, Value: 0.07},
					{Year: 1930, Value: 0.1},
					{Year: 1940, Value: 0.15},
					{Year: 1950, Value: 0.35},
					{Year: 1960, Value: 0.8},
					{Year: 1970, Value: 2.0},
					{Year: 1980, Value: 4.5},
					{Year: 1990, Value: 8.9},
					{Year: 2000, Value: 12.0},
					{Year: 2010, Value: 15.0},
					{Year: 2020, Value: 17.0},
					{Year: 2023, Value: 17.9},
				}
				results[config.ID] = &ChartData{
					ChartConfig: ChartConfig{
						ID:          config.ID,
						Title:       config.Title,
						Description: "The share of land under legal protection has grown from virtually zero to nearly 18%.",
						Source:      config.Source,
						Direction:   config.Direction,
					},
					Year:        2023,
					Value:       &[]float64{17.9}[0],
					ValueColumn: "Protected (%)",
					DataPoints:  protectedLandData,
					LastUpdated: time.Now().Format(time.RFC3339),
				}
			} else {
				// Default placeholder for other manual charts
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
	}

	// Save results
	outputFile := filepath.Join("data", "optimistic_charts.json")
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
