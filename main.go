package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Book struct {
	ID     uint   `json:"id" gorm:"primaryKey"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Genre  string `json:"genre"`
	Tags   string `json:"tags"`
	Cover  string `json:"cover"`
	Description string  `json:"description"`
  	CreatedAt time.Time `json:"created_at"` // auto-filled by GORM
}

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique" json:"username"`
	Email    string `gorm:"unique" json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"` // "user" or "admin"
}

var (
	DB         *gorm.DB
	err        error
	jwtKey     = []byte("supersecretkey")
	refreshKey = []byte("refreshsecretkey")
)

func main() {
	dsn := "root:@tcp(127.0.0.1:3307)/go_crud?charset=utf8mb4&parseTime=True&loc=Local"
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("❌ Failed to connect to MySQL")
	}

	DB.AutoMigrate(&User{}, &Book{})

	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("static/*.html")

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static/index.html")
	})

	r.POST("/register", register)
	r.POST("/login", login)
	r.POST("/refresh", refreshToken)
	r.POST("/logout", logout)

	auth := r.Group("/")
	auth.Use(AuthMiddleware())
	auth.GET("/me", getCurrentUser)
	auth.POST("/books", createBook)
	auth.GET("/books", getBooks)
	auth.GET("/books/:id", getBook)
	auth.PUT("/books/:id", updateBook)
	auth.DELETE("/books/:id", AdminMiddleware(), deleteBook)

	auth.POST("/upload", uploadCover)
	
	fmt.Println("🚀 Server starting on :8080")
	r.Run(":8080")
}

// ------------------------
// User Register & Login
// ------------------------

func register(c *gin.Context) {
	var input User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Trim and validate basic fields
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	if input.Username == "" || input.Password == "" || input.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username, email and password are required"})
		return
	}

	// Check if email already exists
	var existing User
	if err := DB.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already registered"})
		return
	}

	// Check if username already exists
	if err := DB.Where("username = ?", input.Username).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	input.Password = string(hashedPassword)

	if input.Role == "" {
		input.Role = "user"
	}

	if err := DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func login(c *gin.Context) {
	var input User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var user User
	if err := DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Wrong password"})
		return
	}

	// Create JWT access token including role
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	})

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role, // Include role in refresh token too
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})

	accessStr, err := accessToken.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshStr, err := refreshToken.SignedString(refreshKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Set refresh token in cookie (httpOnly)
	c.SetCookie("refresh_token", refreshStr, 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"token": accessStr})
}

func refreshToken(c *gin.Context) {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No refresh token"})
		return
	}

	token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
		return refreshKey, nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	claims := token.Claims.(jwt.MapClaims)
	userID := claims["user_id"]
	// role := claims["role"]

	// Get user from database to ensure they still exist and get current role
	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	newAccess := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    user.Role, // Use current role from database
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	})

	tokenStr, err := newAccess.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new token"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

func logout(c *gin.Context) {
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// ----------------------------
// Middleware
// ----------------------------

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}
		userID := uint(userIDFloat)

		role, ok := claims["role"].(string)
		if !ok {
			// fallback: load from DB if role is missing in token
			var user User
			if err := DB.First(&user, userID).Error; err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
				return
			}
			role = user.Role
		}

		// Store user info in context for handlers to access
		c.Set("user_id", userID)
		c.Set("role", role)

		c.Next()
	}
}

// Middleware to allow only admins
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		c.Next()
	}
}

// ----------------------------
// User info endpoint
// ----------------------------

func getCurrentUser(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := userIDVal.(uint)

	var user User
	if err := DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

// ----------------------------
// Book CRUD
// ----------------------------

func createBook(c *gin.Context) {
	var book Book
	if err := c.ShouldBindJSON(&book); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	book.Title = strings.TrimSpace(book.Title)
	book.Author = strings.TrimSpace(book.Author)
	book.Genre = strings.TrimSpace(book.Genre)
	book.Tags = strings.TrimSpace(book.Tags)
	book.Cover = strings.TrimSpace(book.Cover)
	book.Description = strings.TrimSpace(book.Description)


	if book.Title == "" || book.Author == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title and Author are required"})
		return
	}

	if err := DB.Create(&book).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create book"})
		return
	}

	c.JSON(http.StatusOK, book)
}

func getBooks(c *gin.Context) {
	var books []Book
	search := c.Query("search")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "100")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 100
	}

	offset := (page - 1) * limit
	query := DB.Model(&Book{})

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("title LIKE ? OR author LIKE ? OR genre LIKE ? OR tags LIKE ?", like, like, like, like)
	}

	var total int64
	query.Count(&total)
	
	if err := query.Offset(offset).Limit(limit).Find(&books).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch books"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  books,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func uploadCover(c *gin.Context) {
	file, err := c.FormFile("cover")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form"})
		return
	}

	// Validate file type
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only image files are allowed"})
		return
	}

	// Validate file size (5MB max)
	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size too large (max 5MB)"})
		return
	}

	// Ensure the /static/covers/ directory exists
	saveDir := "./static/covers/"
	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	// Create a safe filename with timestamp to avoid duplicates
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), strings.ReplaceAll(file.Filename, ext, ""), ext)
	filename = filepath.Base(filename) // Security: prevent directory traversal
	savePath := filepath.Join(saveDir, filename)

	// Save file
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"filename": filename})
}

func getBook(c *gin.Context) {
	id := c.Param("id")
	var book Book
	if err := DB.First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}
	c.JSON(http.StatusOK, book)
}



func updateBook(c *gin.Context) {
	id := c.Param("id")

	// Fetch book by ID
	var book Book
	if err := DB.First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	// Read form fields
	title := c.PostForm("title")
	author := c.PostForm("author")
	genre := c.PostForm("genre")
	tags := c.PostForm("tags")
	description := c.PostForm("description") 

	// Validate required fields
	if strings.TrimSpace(title) == "" || strings.TrimSpace(author) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title and Author are required"})
		return
	}

	// Assign updated values
	book.Title = strings.TrimSpace(title)
	book.Author = strings.TrimSpace(author)
	book.Genre = strings.TrimSpace(genre)
	book.Tags = strings.TrimSpace(tags)
	book.Description = strings.TrimSpace(description)
	// Check if a new cover image was uploaded
	file, err := c.FormFile("cover")
	if err == nil {
		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(file.Filename))
		savePath := filepath.Join("static/covers", filename)

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
			return
		}

		book.Cover = filename
	}

	// Save updated book
	if err := DB.Save(&book).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update book"})
		return
	}

	c.JSON(http.StatusOK, book)
}





func deleteBook(c *gin.Context) {
	id := c.Param("id")
	var book Book

	// Step 1: Find the book by ID
	if err := DB.First(&book, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	// Step 2: Delete cover image file if exists
	if book.Cover != "" {
		imagePath := filepath.Join("static/covers", book.Cover)
		if err := os.Remove(imagePath); err != nil && !os.IsNotExist(err) {
			// Log the error, but don't fail the whole operation
			fmt.Printf("⚠️ Failed to delete image: %v\n", err)
		}
	}

	// Step 3: Delete the book from DB
	if err := DB.Delete(&book).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete book"})
		return
	}

	// Step 4: Respond success
	c.JSON(http.StatusOK, gin.H{"message": "✅ Book and cover image deleted successfully"})
}
