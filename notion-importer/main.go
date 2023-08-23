package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

const API_KEY = ""
const DATABASE_ID = ""
const API_URL = "https://api.notion.com/v1"

func loadStoredRecipes(path string) ([]Recipe, error) {
	var recipes []Recipe

	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.Name() == "recipe.json" {
			fileContent, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var recipe Recipe
			err = json.Unmarshal(fileContent, &recipe)
			if err != nil {
				return err
			}

			recipes = append(recipes, recipe)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return recipes, nil
}

func createRow(left, right string) TableChild {
	return TableChild{
		TableRow: TableRow{
			Cells: [][]RichText{
				{
					{
						Type: "text",
						Text: Text{
							Content: left,
						},
					},
				},
				{
					{
						Type: "text",
						Text: Text{
							Content: right,
						},
					},
				},
			},
		},
	}
}

func addPageToDatabase(databaseId string, recipe Recipe) (string, error) {
	textContent := Text{
		Content: recipe.Title,
	}

	titleContents := TitleContent{
		Text: textContent,
	}

	properties := Properties{
		Name: struct {
			Title []TitleContent `json:"title"`
		}{
			Title: []TitleContent{titleContents},
		},
	}

	tableChildren := []TableChild{
		createRow("Menge", "Zutat"),
	}

	for _, ingredient := range recipe.Ingredients {
		tableChildren = append(tableChildren, createRow(ingredient.Amount, ingredient.Name))
	}

	table := Table{
		TableWidth:      2,
		HasColumnHeader: true,
		HasRowHeader:    true,
		Children:        tableChildren,
	}

	page := NotionPageRequestPayload{
		Parent: Parent{DatabaseID: databaseId},
		Props:  properties,
		Children: append([]Child{
			{Object: "block", Type: "table", Table: &table},
		}, createNumberedList(recipe.Steps)...),
	}

	jsonData, err := json.Marshal(page)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", API_URL+"/pages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+API_KEY)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Error adding page to database: %s", (body))
	}

	var response NotionPageResponsePayload
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", err
	}

	return response.ID, nil
}

func main() {
	recipes, err := loadStoredRecipes("../output")
	if err != nil {
		panic(err)
	}

	for _, recipe := range recipes {
		fmt.Println("add recipe: ", recipe.Title)
		id, err := addPageToDatabase(DATABASE_ID, recipe)
		if err != nil {
			panic(err)
		}
		fmt.Println("successfully stored recipe:", id)
	}
}
