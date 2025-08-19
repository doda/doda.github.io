.PHONY: fetch-charts update-charts dev build clean

# Chart data commands
fetch-charts:
	go run cmd/fetch-charts/main.go

update-charts: fetch-charts

# Hugo commands  
dev:
	hugo server -D

build:
	hugo --gc --minify

clean:
	rm -rf public
	rm -rf data/charts