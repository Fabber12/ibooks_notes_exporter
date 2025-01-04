package main

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
	dbThings "ibooks_notes_exporter/db"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	app := &cli.App{
		Name:    "Ibooks notes exporter",
		Usage:   "Export your records from Apple iBooks",
		Authors: []*cli.Author{{Name: "Andrey Korchak", Email: "me@akorchak.software"}},
		Version: "v0.1.0",
		Commands: []*cli.Command{
			{
				Name:   "books",
				Usage:  "Get list of the books with notes and highlights",
				Action: getListOfBooks,
			},
			{
				Name: "version",
				Action: func(context *cli.Context) error {
					fmt.Printf("%s\n", context.App.Version)
					return nil
				},
			},
			{
				Name:      "export",
				HideHelp:  false,
				Usage:     "Export all notes and highlights from book with [BOOK_ID] to the specified directory",
				UsageText: "Export all notes and highlights from book with [BOOK_ID]",
				Action:    exportNotesAndHighlights,
				ArgsUsage: "ibooks_notes_exporter export BOOK_ID_GOES_HERE --output PATH_TO_DIRECTORY",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "book_id",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "output",
						Usage:    "Directory where the output files will be saved",
						Required: true,
					},
					&cli.IntFlag{
						Name:     "skip_first_x_notes",
						Value:    0,
						Required: false,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func getListOfBooks(cCtx *cli.Context) error {
	db := dbThings.GetDBConnection()

	rows, err := db.Query(dbThings.GetAllBooksDbQueryConstant)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"SingleBook ID", "# notes", "Title and Author"})

	var singleBook dbThings.SingleBookInList
	for rows.Next() {
		err := rows.Scan(&singleBook.Id, &singleBook.Title, &singleBook.Author, &singleBook.Number)
		if err != nil {
			log.Fatal(err)
		}
		truncatedTitle := singleBook.Title
		if len(singleBook.Title) > 30 {
			truncatedTitle = singleBook.Title[:30] + "..."
		}
		standardizedAuthor := GetLastNames(singleBook.Author)
		t.AppendRows([]table.Row{
			{singleBook.Id, singleBook.Number, fmt.Sprintf("%s %s", truncatedTitle, standardizedAuthor)},
		})
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	t.Render()
	return nil
}

func exportNotesAndHighlights(cCtx *cli.Context) error {
	db := dbThings.GetDBConnection()
	defer db.Close()

	bookId := cCtx.String("book_id")
	skipXNotes := cCtx.Int("skip_first_x_notes")
	outputDir := cCtx.String("output")

	// Ensure the output directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println(bookId)

	var book dbThings.SingleBook
	row := db.QueryRow(dbThings.GetBookDataById, bookId)
	err := row.Scan(&book.Name, &book.Author)
	if err != nil {
		log.Println(err)
		log.Fatal("SingleBook is not found in iBooks!")
	}

	rows, err := db.Query(dbThings.GetNotesHighlightsById, bookId, skipXNotes)
	if err != nil {
		log.Fatal(err)
	}

	colorFiles := make(map[string]*os.File)
	var singleHighlightNote dbThings.SingleHighlightNote
	for rows.Next() {
		err := rows.Scan(&singleHighlightNote.HightLight, &singleHighlightNote.Note, &singleHighlightNote.Style, &singleHighlightNote.IsUnderline)
		if err != nil {
			log.Fatal(err)
		}

		color := getColorName(singleHighlightNote.Style)
		typeAnnotation := "Evidenziatura"
		if singleHighlightNote.IsUnderline == 1 {
			typeAnnotation = "Sottolineatura"
		}

		// Open or create a file for the specific color/type in the specified output directory
		fileName := filepath.Join(outputDir, fmt.Sprintf("%s_%s.md", book.Name, color))
		if colorFiles[fileName] == nil {
			file, err := os.Create(fileName)
			if err != nil {
				log.Fatal(err)
			}
			colorFiles[fileName] = file
		}

		file := colorFiles[fileName]
		if typeAnnotation == "Sottolineatura" {
			fmt.Fprintf(file, "%s\n\n", strings.Replace(singleHighlightNote.HightLight, "\n", "", -1))
		} else {
			// Apply color using markdown syntax for color
			fmt.Fprintf(file, "<span style=\"color:%s\">%s</span>\n\n", getColorHex(color), strings.Replace(singleHighlightNote.HightLight, "\n", "", -1))
		}
	}

	// Close all files
	for _, file := range colorFiles {
		file.Close()
	}

	return nil
}

func getColorName(style int) string {
	switch style {
	case 1:
		return "Verde"
	case 2:
		return "Blu"
	case 3:
		return "Giallo"
	case 4:
		return "Rosa"
	case 5:
		return "Viola"
	default:
		return "Sottolineato"
	}
}

func getColorHex(color string) string {
	switch color {
	case "Blu":
		return "#0077ff"
	case "Giallo":
		return "#ffb700"
	case "Verde":
		return "#67e600"
	case "Rosa":
		return "#f56ca3"
	case "Viola":
		return "#c603fc"
	default:
		return "#000000"
	}
}

func GetLastNames(names string) string {
	nameList := strings.Split(names, " & ")
	if len(nameList) == 1 {
		return GetLastName(nameList[0])
	}
	if len(nameList) == 2 {
		return GetLastName(nameList[0]) + " & " + GetLastName(nameList[1])
	}
	firstName := nameList[0]
	lastNames := make([]string, len(nameList)-1)
	for i, name := range nameList[1:] {
		lastNames[i] = GetLastName(name)
	}
	return GetLastName(firstName) + " & " + strings.Join(lastNames, " & ")
}

func GetLastName(name string) string {
	words := strings.Fields(name)
	var lastName string
	for i := len(words) - 1; i >= 0; i-- {
		if !isHonorific(words[i]) {
			lastName = words[i]
			break
		}
	}
	lastName = strings.TrimSuffix(lastName, ",")
	lastName = strings.TrimSuffix(lastName, ".")
	return "(" + lastName + ")"
}

func isHonorific(word string) bool {
	return len(word) <= 3 && (word[len(word)-1] == '.' || word[len(word)-1] == ',')
}
