package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"shoes/helper"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Shoe struct {
	gorm.Model
	Name     string
	Sku      string
	ImageURL string
}

type Store struct {
	gorm.Model
	Name string
}

type PriceHistory struct {
	gorm.Model
	ShoeID    uint
	StoreID   uint
	PriceDate time.Time
	Price     int
}

func get_shoe_data() ([]helper.PageData, []Shoe) {
	total_shoes := 200
	var shoes []Shoe
	var pages []helper.PageData
	var pageData helper.PageData

	for p := 1; p < total_shoes/50; p++ {
		url := fmt.Sprintf("https://ac.cnstrc.com/search/dunk?c=ciojs-client-2.29.12&key=key_XT7bjdbvjgECO5d8&i=0813ce0b-c5a4-4911-af43-67aafa925cd9&s=2&page=%d&num_results_per_page=50&sort_by=relevance&sort_order=descending&fmt_options%%5Bhidden_fields%%5D=gp_lowest_price_cents_3&fmt_options%%5Bhidden_fields%%5D=gp_instant_ship_lowest_price_cents_3&fmt_options%%5Bhidden_facets%%5D=gp_lowest_price_cents_3&fmt_options%%5Bhidden_facets%%5D=gp_instant_ship_lowest_price_cents_3&_dt=%d", p, time.Now().Unix())

		resp, err := http.Get(url)
		if err != nil {
			panic(err)
		}

		decoder := json.NewDecoder(resp.Body)

		error := decoder.Decode(&pageData)
		if error != nil {
			println(p)
			// panic(error)
		}

		pages = append(pages, pageData)
		shoes = append(shoes, (convert_page_to_shoes(pageData))...)

		if p == 1 {
			total_shoes = pageData.Response.TotalNumResults
		}
		// time.Sleep(1 * time.Second)
	}

	return pages, shoes
}

func get_store_data() helper.StoreData {
	url := fmt.Sprintf("https://ac.cnstrc.com/autocomplete/dunk?c=ciojs-client-2.29.12&key=key_XT7bjdbvjgECO5d8&i=0813ce0b-c5a4-4911-af43-67aafa925cd9&s=1&num_results_Products=50&_dt=%d", time.Now().Unix())

	var storeData helper.StoreData
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(resp.Body)

	error := decoder.Decode(&storeData)
	if error != nil {
		panic(error)
	}

	return storeData
}

func get_all_shoes(db *gorm.DB) []Shoe {
	var shoes []Shoe
	db.Find(&shoes)
	return shoes
}

func get_shoes_map(db *gorm.DB) map[string]int {
	shoes := get_all_shoes(db)
	shoes_map := make(map[string]int)
	for _, shoe := range shoes {
		shoes_map[shoe.Sku] = int(shoe.ID)
	}
	return shoes_map
}

func create_new_shoes(db *gorm.DB, shoes []Shoe) {
	db.CreateInBatches(shoes, 100)
}

func convert_page_to_shoes(pageData helper.PageData) []Shoe {
	var shoes []Shoe
	for _, p := range pageData.Response.Results {
		shoe := Shoe{Name: p.Value, Sku: p.Data.Sku, ImageURL: p.Data.ImageURL}
		shoes = append(shoes, shoe)
	}
	return shoes
}

func convert_data_to_shoes(data helper.StoreData) []Shoe {
	var shoes []Shoe
	for _, d := range data.Sections.Products {
		shoe := Shoe{Name: d.Value, Sku: d.Data.Sku}
		shoes = append(shoes, shoe)
	}
	return shoes
}

func find_missing_shoes(shoes_map map[string]int, shoes []Shoe) []Shoe {
	var missing_shoes []Shoe
	for _, shoe := range shoes {
		if _, ok := shoes_map[shoe.Sku]; !ok {
			missing_shoes = append(missing_shoes, shoe)
		}
	}
	return missing_shoes
}

func db() *gorm.DB {
	dsn := "host=localhost user=postgres password=pass dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&Shoe{})
	db.AutoMigrate(&Store{})
	db.AutoMigrate(&PriceHistory{})

	// db.Create(&Store{Name: "Goat"})

	// db.Create(&Shoe{Name: "Nike"})
	// db.Create(&Shoe{Name: "Adidas"})

	return db
}

func convert_pages_to_price_history(pages []helper.PageData, shoe_map map[string]int) []PriceHistory {
	var price_histories []PriceHistory
	for _, p := range pages {
		for _, r := range p.Response.Results {
			price := PriceHistory{Price: r.Data.LowestPriceCents, PriceDate: time.Now(), StoreID: 1, ShoeID: uint(shoe_map[r.Data.Sku])}
			price_histories = append(price_histories, price)
		}
	}
	return price_histories
}

func convert_data_to_price_history(data helper.StoreData, shoe_map map[string]int) []PriceHistory {
	var price_histories []PriceHistory
	for _, d := range data.Sections.Products {
		price := PriceHistory{Price: d.Data.LowestPriceCents, PriceDate: time.Now(), StoreID: 1, ShoeID: uint(shoe_map[d.Data.Sku])}
		price_histories = append(price_histories, price)
	}
	return price_histories
}

func add_price_histories(db *gorm.DB, price_histories []PriceHistory) {
	db.CreateInBatches(price_histories, 100)
}

func main() {
	db := db()
	shoes_map := get_shoes_map(db)
	pages, shoes := get_shoe_data()

	// store_data := get_store_data()
	// shoes := convert_data_to_shoes(store_data)

	missing_shoes := find_missing_shoes(shoes_map, shoes)
	if len(missing_shoes) > 0 {
		create_new_shoes(db, missing_shoes)
		for _, shoe := range missing_shoes {
			shoes_map[shoe.Sku] = int(shoe.ID)
		}
	}

	price_histories := convert_pages_to_price_history(pages, shoes_map)
	add_price_histories(db, price_histories)
}
