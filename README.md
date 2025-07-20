 📚 Book Manager

A simple, powerful book management web app built with Go (Gin), Bootstrap 5, and vanilla JavaScript.

## 🔧 Features

- ✅ Add, edit, delete books
- 🔍 Search by title/author
- 🎭 Filter by genre
- 🔠 Sort by title or author (A–Z / Z–A)
- 🏷️ Tag-based filtering
- 📷 Full-size cover preview modal
- 🧼 Clear filters button
- 📌 Highlight search matches
- 📦 Export as JSON or CSV
- 🌙 Dark mode toggle
- ✍️ Book description field
- 🕒 Created-at timestamps

## 🛠️ Tech Stack

- **Backend**: Go + Gin
- **Frontend**: HTML, Bootstrap 5, JS
- **Database**: GORM (SQLite/MySQL/PostgreSQL supported)
- **Authentication**: JWT-based (optional)

## 📁 Folder Structure

book-manager/
├── static/
│ └── covers/ # Uploaded book cover images
├── templates/ # HTML files
├── main.go # Entry point
├── models.go # DB models
├── handlers.go # API logic
├── auth.go # (If auth used)
├── go.mod
└── README.md



## 🚀 Getting Started

### 1. Clone the repo

```bash
git clone https://github.com/shebinbalan/go-my-sql-crud-first-project.git
cd book-manager
2. Run the backend

go run main.go
By default, it runs at:
📍 http://localhost:8080

3. Visit frontend
Open index.html in your browser
(or serve via backend templates)

📝 API Endpoints
Method	Endpoint	Description
GET	/books	List all books
GET	/books/:id	Get book by ID
POST	/books	Create book (multipart)
PUT	/books/:id	Update book (multipart)
DELETE	/books/:id	Delete book


Built with ❤️ by Shebin


