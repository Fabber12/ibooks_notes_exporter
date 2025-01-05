package main

import (
	"database/sql"
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
		Version: "v0.3.3",
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
				Usage:     "Export all notes and highlights from book with [BOOK_ID] to Markdown files",
				UsageText: "Export all notes and highlights from book with [BOOK_ID]",
				Action:    exportNotesAndHighlights,
				ArgsUsage: "ibooks_notes_exporter export BOOK_ID_GOES_HERE",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "book_id",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "output",
						Usage:    "Output directory for Markdown files",
						Value:    "output",
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
	outputDir := cCtx.String("output")

	// Ensure the output directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}

	notesFile, err := os.Create(fmt.Sprintf("%s/ibooks_extracted_notes.md", outputDir))
	if err != nil {
		log.Fatal(err)
	}
	defer notesFile.Close()

	vocabFile, err := os.Create(fmt.Sprintf("%s/ibooks_vocabulary.md", outputDir))
	if err != nil {
		log.Fatal(err)
	}
	defer vocabFile.Close()

	var book dbThings.SingleBook
	row := db.QueryRow(dbThings.GetBookDataById, bookId)
	err = row.Scan(&book.Name, &book.Author)
	if err != nil {
		log.Println(err)
		log.Fatal("SingleBook is not found in iBooks!")
	}

	fmt.Fprintf(notesFile, "# %s – Estratto delle evidenziazioni\n\n", book.Name)
	fmt.Fprintf(notesFile, "<!--\nT = Titolo: Indica un capitolo, sezione o sottosezione del libro.\nD = Nota discorsiva: Spiegazioni estese, metafore o esempi che arricchiscono il testo.\nP = Termine tecnico: Concetti o termini specialistici rilevanti per il contenuto.\nN = Nota importante: Concetti chiave o informazioni cruciali da ricordare.\nU = Sottolineatura significativa: Frasi o concetti evidenziati per il loro impatto o rilevanza.\n-->\n\n")
	fmt.Fprintf(vocabFile, "# %s – Vocabolario\n\n", book.Name)

	rows, err := db.Query(dbThings.GetNotesHighlightsByIdWithContext, bookId, 0)
	if err != nil {
		log.Fatal(err)
	}

	index := 1
	vocabIndex := 1
	for rows.Next() {
		var highlight sql.NullString
		var note sql.NullString
		var context sql.NullString
		var style, isUnderline int
		if err := rows.Scan(&highlight, &note, &context, &style, &isUnderline); err != nil {
			log.Fatal(err)
		}

		if !highlight.Valid {
			continue
		}

		label := classifyHighlight(style, isUnderline)
		if label == "[Y]" {
			// Write yellow highlights to vocabulary file
			fmt.Fprintf(vocabFile, "%d. %s\n", vocabIndex, strings.Replace(highlight.String, "\n", "", -1))
			if context.Valid {
				fmt.Fprintf(vocabFile, "   Sentence: %s\n\n", strings.Replace(context.String, "\n", "", -1))
			}
			vocabIndex++
			continue
		}

		// Write other highlights to the notes file
		fmt.Fprintf(notesFile, "%d. %s \"%s\"\n\n", index, label, strings.Replace(highlight.String, "\n", "", -1))
		if note.Valid {
			fmt.Fprintf(notesFile, "\tNota: %s\n\n", note.String)
		}
		index++
	}

	return nil
}

func classifyHighlight(style, isUnderline int) string {
	if isUnderline == 1 {
		return "[U]"
	}
	switch style {
	case 1:
		return "[P]" // Verde - Punto pratico
	case 2:
		return "[D]" // Blu - Definizione
	case 3:
		return "[Y]" // Giallo - Vocaboli
	case 4:
		return "[N]" // Rosa - Nota generale
	case 5:
		return "[T]" // Viola - Titoli
	default:
		return "[U]" // Sottolineato
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
		if len(words[i]) > 0 {
			lastName = words[i]
			break
		}
	}
	lastName = strings.TrimSuffix(lastName, ",")
	lastName = strings.TrimSuffix(lastName, ".")
	return lastName
}