package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type UrlSet struct {
	XMLName xml.Name `xml:"urlset"`
	Urls    []Url    `xml:"url"`
}

type Url struct {
	Loc string `xml:"loc"`
}

type Ingredients struct {
	Amount string
	Name   string
}

type Recipe struct {
	Title       string
	Subtitle    string
	Labels      []string
	Tags        []string
	Ingredients []Ingredients
	Steps       []string
}

func getRecipeUrls(tld string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("https://www.hellofresh.%s/sitemap_recipe_pages.xml", tld))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var urlSet UrlSet
	err = xml.Unmarshal(body, &urlSet)
	if err != nil {
		return nil, err
	}

	var urls []string
	for _, url := range urlSet.Urls {
		urls = append(urls, url.Loc)
	}

	return urls, nil
}

func fetchHtml(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 status code: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func parseHtml(html string) (Recipe, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return Recipe{}, err
	}

	descriptionSelection := doc.Find("[data-test-id=recipe-description]").First()
	title := descriptionSelection.Find("h1").First().Text()
	subtitle := descriptionSelection.Find("h2").First().Text()

	labels := make([]string, 0)
	doc.Find("[data-test-id=label-text] span").Each(func(i int, s *goquery.Selection) {
		labels = append(labels, s.Text())
	})

	tags := make([]string, 0)
	doc.Find("[data-test-id=recipe-description-tag] span").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if text != "•" {
			tags = append(tags, text)
		}
	})

	ingredientSelection := doc.Find("[data-test-id=ingredient-item-shipped], [data-test-id=ingredient-item-not-shipped]")
	var ingredients []Ingredients

	ingredientSelection.Each(func(i int, s *goquery.Selection) {
		container := s.Find("div").Last()
		amount := container.Find("p").First().Text()
		name := container.Find("p").Last().Text()

		ingredients = append(ingredients, Ingredients{
			Amount: amount,
			Name:   name,
		})
	})

	var steps []string
	doc.Find("[data-test-id=instruction-step]").Each(func(i int, s *goquery.Selection) {
		container := s.Find("div").Last()
		step := container.Find("p").First().Text()

		steps = append(steps, step)
	})

	return Recipe{
		Title:       title,
		Subtitle:    subtitle,
		Labels:      labels,
		Tags:        tags,
		Ingredients: ingredients,
		Steps:       steps,
	}, nil
}

func storeRecipeAsJSON(recipe Recipe, filename string, path string) error {
	jsonData, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return err
	}

	err = os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(fmt.Sprintf("%s/%s.json", path, filename), jsonData, 0644)
	return err
}

func getFileNameFromUrl(url string) string {
	return path.Base(url)
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func getAndSaveRecipe(url string, path string, fileName string) error {
	targetPath := fmt.Sprintf("%s/%s.json", path, fileName)
	exists, err := fileExists(targetPath)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("file already exists: %s", targetPath)
	}

	htmlContent, err := fetchHtml(url)
	if err != nil {
		return err
	}

	recipe, err := parseHtml(htmlContent)
	if err != nil {
		return err
	}

	err = storeRecipeAsJSON(recipe, fileName, path)
	if err != nil {
		return err
	}

	return nil
}

func getAndSaveAllRecipes(urls []string, delay time.Duration, path string) {
	for _, url := range urls {
		fileName := getFileNameFromUrl(url)
		err := getAndSaveRecipe(url, path, fileName)
		if err != nil {
			fmt.Println("Error fetching recipes:", err)
			continue
		}
		fmt.Println("Fetched recipe:", fileName)
		time.Sleep(delay)
	}

}

func main() {
	outputPath := flag.String("output", "./output", "Path to store the output")
	delay := flag.Int("delay", 5000, "Delay per request in ms")
	locale := flag.String("tld", "de", "Top level domain of the HelloFresh website")

	flag.Parse()

	delayDuration := time.Duration(*delay) * time.Millisecond
	recipeUrls, err := getRecipeUrls(*locale)
	if err != nil {
		fmt.Println("Error fetching URLs:", err)
		return
	}

	getAndSaveAllRecipes(recipeUrls, delayDuration, *outputPath)
}
