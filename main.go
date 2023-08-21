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
	PdfUrl      string
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
	pdfUrl := doc.Find("[data-test-id=recipe-pdf]").First().AttrOr("href", "")

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
		name := container.Find("p").Eq(1).Text()

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
		PdfUrl:      pdfUrl,
		Labels:      labels,
		Tags:        tags,
		Ingredients: ingredients,
		Steps:       steps,
	}, nil
}

func storeRecipeAsJSON(recipe Recipe, targetPath string) error {
	fmt.Print("STORE:", targetPath)
	jsonData, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(targetPath, jsonData, 0644)
	return err
}

func downloadPDFFromURL(pdfURL string, targetPath string) error {
	resp, err := http.Get(pdfURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	outFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func getBaseFromUrl(url string) string {
	return path.Base(url)
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func getAndSaveRecipe(url string, path string, recipeName string, downloadPdf bool) error {
	targetDir := fmt.Sprintf("%s/%s", path, recipeName)

	targetRecipePath := fmt.Sprintf("%s/recipe.json", targetDir)
	targetRecipePdf := fmt.Sprintf("%s/recipe.pdf", targetDir)

	exists, err := pathExists(targetDir)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("dir already exists: %s", targetDir)
	}

	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		return err
	}

	htmlContent, err := fetchHtml(url)
	if err != nil {
		return err
	}

	recipe, err := parseHtml(htmlContent)
	if err != nil {
		return err
	}

	err = storeRecipeAsJSON(recipe, targetRecipePath)
	if err != nil {
		return err
	}

	if downloadPdf && recipe.PdfUrl != "" {
		err := downloadPDFFromURL(recipe.PdfUrl, targetRecipePdf)
		if err != nil {
			return err
		}
	}

	return nil
}

func getAndSaveAllRecipes(urls []string, delay time.Duration, path string) {
	for i, url := range urls {
		recipeName := getBaseFromUrl(url)
		err := getAndSaveRecipe(url, path, recipeName, false)
		if err != nil {
			fmt.Println("Error fetching recipes:", err)
			continue
		}
		fmt.Printf("%d/%d Fetched recipe: %s \n", i, len(urls)+1, recipeName)
		time.Sleep(delay)
	}
}

func uniqueStrings(strings []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, s := range strings {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
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

	getAndSaveAllRecipes(uniqueStrings(recipeUrls), delayDuration, *outputPath)
}
