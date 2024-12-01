package main

import (
    "database/sql"
    "log"
    "net/http"

    _ "github.com/go-sql-driver/mysql" // MySQL driver
    "github.com/gin-gonic/gin"
    "github.com/markbates/goth"
    "github.com/markbates/goth/gothic"
    "github.com/markbates/goth/providers/google"
)

var db *sql.DB // Global database connection

// Initialize the database connection
func connectDatabase() {
    var err error
    dsn := "root:saqqi_king_0.1@tcp(127.0.0.1:3306)/hackathon" // MySQL connection string
    db, err = sql.Open("mysql", dsn)
    if err != nil {
        log.Fatal("Failed to connect to the database:", err)
    }

    // Verify the connection
    if err := db.Ping(); err != nil {
        log.Fatal("Database is unreachable:", err)
    }

    log.Println("Connected to the database successfully!")
}

func main() {
    // Initialize database connection
    connectDatabase()

    // Set up Google OAuth provider
    goth.UseProviders(
        google.New(
            "122117947430-dqeqo5h1va6bpcv72rcptpt7f92cufko.apps.googleusercontent.com", // Replace with your Client ID
            "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",                                     // Replace with your Client Secret
            "http://localhost:8080/auth/google/callback",
        ),
    )

    gothic.GetProviderName = func(req *http.Request) (string, error) {
        provider := req.URL.Query().Get("provider")
        if provider == "" {
            provider = "google" // Default to Google
        }
        return provider, nil
    }

    // Initialize Gin router
    r := gin.Default()

    // Google OAuth routes
    r.GET("/auth/:provider", func(c *gin.Context) {
        gothic.BeginAuthHandler(c.Writer, c.Request)
    })


    r.GET("/auth/:provider/callback", func(c *gin.Context) {
        user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        // Save user data into MySQL
        query := "INSERT INTO users (id, email, name, profile_picture) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE name = VALUES(name), profile_picture = VALUES(profile_picture)"
        _, err = db.Exec(query, user.UserID, user.Email, user.Name, user.AvatarURL)
        if err != nil {
            log.Println("Error saving user data:", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
            return
        }

        c.JSON(http.StatusOK, gin.H{"user": user})
    })

    // Notes API routes

    // Create a new note
    r.POST("/notes", func(c *gin.Context) {
        var note struct {
            UserID  string `json:"user_id"`
            Title   string `json:"title"`
            Content string `json:"content"`
        }
        if err := c.ShouldBindJSON(&note); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        query := "INSERT INTO notes (id,user_id, title, content) VALUES (UUID(),?, ?, ?)"
        _, err := db.Exec(query, note.UserID, note.Title, note.Content)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"message": "Note created successfully!"})
    })

    // Retrieve all notes for a user
    r.GET("/notes/:user_id", func(c *gin.Context) {
        userID := c.Param("user_id")

        rows, err := db.Query("SELECT id, title, content, created_at FROM notes WHERE user_id = ?", userID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        defer rows.Close()

        var notes []map[string]interface{}
        for rows.Next() {
            var id, title, content, createdAt string
            rows.Scan(&id, &title, &content, &createdAt)
            notes = append(notes, map[string]interface{}{
                "id":        id,
                "title":     title,
                "content":   content,
                "created_at": createdAt,
            })
        }

        c.JSON(http.StatusOK, gin.H{"notes": notes})
    })

    // Update a note
    r.PUT("/notes/:id", func(c *gin.Context) {
        noteID := c.Param("id")
        var note struct {
            Title   string `json:"title"`
            Content string `json:"content"`
        }
        if err := c.ShouldBindJSON(&note); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }

        query := "UPDATE notes SET title = ?, content = ? WHERE id = ?"
        _, err := db.Exec(query, note.Title, note.Content, noteID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        c.JSON(http.StatusOK, gin.H{"message": "Note updated successfully!"})
    })

    // Delete a note
    r.DELETE("/notes/:id", func(c *gin.Context) {
        noteID := c.Param("id")

        query := "DELETE FROM notes WHERE id = ?"
        _, err := db.Exec(query, noteID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        c.JSON(http.StatusOK, gin.H{"message": "Note deleted successfully!"})
    })

    // Start the server
    log.Println("Server running on http://localhost:8080")
    r.Run(":8080")
}
