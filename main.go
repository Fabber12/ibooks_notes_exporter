package main

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
	dbThings "ibooks_notes_exporter/db"
	"log"
	"os"
	"strings"
)

func main() {
	app := &cli.App{
		Name:    "Ibooks notes exporter",
		Usage:   "Export your records from Apple iBooks",
		Authors: []*cli.Author{{Name: "Andrey Korchak", Email: "me@akorchak.software"}},
		Version: "v0.0.6",
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
				Usage:     "Export all notes and highlights from book with [BOOK_ID]",
				UsageText: "Export all notes and highlights from book with [BOOK_ID]",
				Action:    exportNotesAndHighlights,
				ArgsUsage: "ibooks_notes_exporter export BOOK_ID_GOES_HERE",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "book_id",
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
	fmt.Println(bookId)

	var book dbThings.SingleBook
	row := db.QueryRow(dbThings.GetBookDataById, bookId)
	err := row.Scan(&book.Name, &book.Author)
	if err != nil {
		log.Println(err)
		log.Fatal("SingleBook is not found in iBooks!")
	}

	fmt.Println(fmt.Sprintf("# %s — %s\n", book.Name, book.Author))

	rows, err := db.Query(dbThings.GetNotesHighlightsById, bookId, skipXNotes)
	if err != nil {
		log.Fatal(err)
	}

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

		fmt.Printf("> %s\n\n", strings.Replace(singleHighlightNote.HightLight, "\n", "", -1))
		fmt.Printf("_Tipo: %s | Colore: %s_\n\n", typeAnnotation, color)

		if singleHighlightNote.Note.Valid {
			fmt.Printf("%s\n\n", strings.Replace(singleHighlightNote.Note.String, "\n", "", -1))
		}

		fmt.Println("---\n\n")
	}

	return nil
}

func getColorName(style int) string {
	switch style {
	case 1:
		return "Blu"
	case 2:
		return "Giallo"
	case 3:
		return "Verde"
	case 4:
		return "Rosa"
	case 5:
		return "Viola"
	default:
		return "Sconosciuto"
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
