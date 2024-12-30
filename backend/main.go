package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

type Note struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

// Fungsi untuk menghubungkan ke database
func connect() (*sql.DB, error) {
	// Membaca password dari Docker secret
	bin, err := os.ReadFile("/run/secrets/db-password")
	if err != nil {
		return nil, err
	}
	return sql.Open("postgres", fmt.Sprintf("postgres://postgres:%s@db:5432/example?sslmode=disable", string(bin)))
}

// Menampilkan halaman utama dengan daftar catatan
func homeHandler(c echo.Context) error {
	db, err := connect()
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to connect to database")
	}
	defer db.Close()

	// Mengambil catatan dari database dengan urutan terbaru
	rows, err := db.Query("SELECT id, content FROM notes ORDER BY id DESC")
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to fetch notes")
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.Content); err != nil {
			log.Println(err)
			return c.String(http.StatusInternalServerError, "Failed to read note")
		}
		notes = append(notes, note)
	}

	// Memuat template dari file
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to load template")
	}

	// Menyajikan template dengan data catatan
	return tmpl.Execute(c.Response().Writer, notes)
}

// Menambahkan catatan baru
func addNoteHandler(c echo.Context) error {
	if c.Request().Method == http.MethodPost {
		content := c.FormValue("content")

		db, err := connect()
		if err != nil {
			log.Println(err)
			return c.String(http.StatusInternalServerError, "Failed to connect to database")
		}
		defer db.Close()

		// Menambahkan catatan ke dalam database
		_, err = db.Exec("INSERT INTO notes (content) VALUES ($1)", content)
		if err != nil {
			log.Println(err)
			return c.String(http.StatusInternalServerError, "Failed to add note")
		}

		// Redirect ke halaman utama
		return c.Redirect(http.StatusSeeOther, "/")
	}
	return nil
}

// Menghapus catatan berdasarkan ID
func deleteNoteHandler(c echo.Context) error {
	// Mendapatkan id dari URL
	id := c.Param("id")

	// Cek jika id kosong atau tidak valid
	if id == "" {
		return c.String(http.StatusBadRequest, "Invalid ID")
	}

	db, err := connect()
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to connect to database")
	}
	defer db.Close()

	// Menghapus catatan dari database
	_, err = db.Exec("DELETE FROM notes WHERE id = $1", id)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "Failed to delete note")
	}

	// Redirect ke halaman utama
	return c.Redirect(http.StatusSeeOther, "/")
}

// Persiapan tabel database (Jika belum ada)
func prepare() error {
	db, err := connect()
	if err != nil {
		return err
	}
	defer db.Close()

	// Memastikan tabel notes ada
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS notes (id SERIAL PRIMARY KEY, content TEXT)"); err != nil {
		return err
	}

	return nil
}

func main() {
	// Memastikan database sudah siap
	log.Print("Preparing database...")
	if err := prepare(); err != nil {
		log.Fatal(err)
	}

	// Inisialisasi Echo
	e := echo.New()

	// Menentukan route untuk halaman utama, menambahkan catatan, dan menghapus catatan
	e.GET("/", homeHandler)
	e.POST("/add", addNoteHandler)
	e.POST("/delete/:id", deleteNoteHandler)

	// Menambahkan middleware untuk logging
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			log.Printf("%s %s", c.Request().Method, c.Request().URL)
			return next(c)
		}
	})

	// Menjalankan server pada port 8000
	log.Print("Server is running on port 8000")
	e.Logger.Fatal(e.Start(":8000"))
}
