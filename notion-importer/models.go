package main

import "strings"

type Ingredients struct {
	Amount string
	Name   string
}

type Recipe struct {
	Title       string
	Subtitle    string
	CoverImg    string
	PdfUrl      string
	Labels      []string
	Tags        []string
	Ingredients []Ingredients
	Steps       []string
}

type Text struct {
	Content string `json:"content"`
}

type TitleContent struct {
	Text Text `json:"text"`
}

type Properties struct {
	Name struct {
		Title []TitleContent `json:"title"`
	} `json:"Name"`
}

type Parent struct {
	DatabaseID string `json:"database_id"`
}

type Table struct {
	TableWidth      int64        `json:"table_width"`
	HasColumnHeader bool         `json:"has_column_header"`
	HasRowHeader    bool         `json:"has_row_header"`
	Children        []TableChild `json:"children"`
}

type RichText struct {
	Type string `json:"type"`
	Text Text   `json:"text"`
}

type TableChild struct {
	TableRow TableRow `json:"table_row"`
}

type TableRow struct {
	Cells [][]RichText `json:"cells"`
}

type NumberedListItem struct {
	RichText []RichText `json:"rich_text"`
}

type Child struct {
	Object           string            `json:"object"`
	Type             string            `json:"type"`
	Table            *Table            `json:"table,omitempty"`
	NumberedListItem *NumberedListItem `json:"numbered_list_item,omitempty"`
}

type NotionPageRequestPayload struct {
	Parent   Parent     `json:"parent"`
	Props    Properties `json:"properties"`
	Children []Child    `json:"children"`
}

type NotionPageResponsePayload struct {
	ID string `json:"id"`
}

func createNumberedList(items []string) []Child {
	var blocks []Child

	for _, item := range items {
		block := Child{
			Object: "block",
			Type:   "numbered_list_item",
			NumberedListItem: &NumberedListItem{
				RichText: []RichText{
					{
						Type: "text",
						Text: Text{
							Content: strings.ReplaceAll(item, "\n", " "),
						},
					},
				},
			},
		}
		blocks = append(blocks, block)
	}
	return blocks
}
