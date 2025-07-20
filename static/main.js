// ✅ main.js (extracted from your HTML)

let allBooks = [];
let currentPage = 1;
let itemsPerPage = 5;
let bookToDelete = null;
let currentUser = null;

const darkModeToggle = document.getElementById("darkModeToggle");
darkModeToggle.addEventListener("change", function () {
  document.body.classList.toggle("dark-mode", this.checked);
  localStorage.setItem("darkMode", this.checked ? "on" : "off");
  updateInputsStyle();
});

function applyDarkModePreference() {
  const mode = localStorage.getItem("darkMode");
  const enabled = mode === "on";
  document.body.classList.toggle("dark-mode", enabled);
  darkModeToggle.checked = enabled;
  updateInputsStyle();
}

function updateInputsStyle() {
  const input = document.querySelector("#searchInput");
  if (document.body.classList.contains("dark-mode")) {
    input.classList.add("dark-mode");
  } else {
    input.classList.remove("dark-mode");
  }
}

function getAccessToken() {
  return localStorage.getItem("access_token");
}

function getAuthHeaders() {
  return {
    "Content-Type": "application/json",
    "Authorization": "Bearer " + getAccessToken()
  };
}

async function refreshToken() {
  try {
    const res = await fetch("/refresh", { method: "POST", credentials: "include" });
    const data = await res.json();
    if (data.token) {
      localStorage.setItem("access_token", data.token);
      return true;
    }
  } catch {}
  return false;
}

function fetchWithRefresh(url, options) {
  return fetch(url, { ...options, credentials: "include" }).then(async res => {
    if (res.status === 401) {
      const refreshed = await refreshToken();
      if (refreshed) {
        options.headers["Authorization"] = "Bearer " + getAccessToken();
        return fetch(url, { ...options, credentials: "include" });
      } else {
        throw new Error("Session expired");
      }
    }
    return res;
  });
}

function fetchBooks() {
  fetchWithRefresh("/books", {
    method: "GET",
    headers: getAuthHeaders()
  })
    .then(res => res.json())
    .then(data => {
      allBooks = data.data || data;
      renderBooks();
      renderPagination();
      populateGenreFilter(); 

    })
    .catch(() => {
      alert("❌ Session expired. Please login again.");
      logout();
    });
}

function fetchCurrentUser() {
  fetchWithRefresh("/me", {
    method: "GET",
    headers: getAuthHeaders()
  })
    .then(res => res.json())
    .then(data => {
      currentUser = data;
      document.getElementById("userInfo").innerText = `👤 ${data.username} (${data.role})`;
      fetchBooks();
    })
    .catch(() => {
      logout();
    });
}



// function renderBooks() {
//   const tbody = document.getElementById("bookTable");
//   tbody.innerHTML = "";

//   const search = document.getElementById("searchInput").value.toLowerCase();
//   const selectedGenre = document.getElementById("genreFilter").value.toLowerCase(); // ✅ new line

//   const filtered = allBooks.filter(b =>
//     (b.title.toLowerCase().includes(search) || b.author.toLowerCase().includes(search)) &&
//     (selectedGenre === "" || (b.genre && b.genre.toLowerCase() === selectedGenre)) // ✅ updated filter
//   );

//   const start = (currentPage - 1) * itemsPerPage;
//   const end = start + itemsPerPage;
//   const pageBooks = filtered.slice(start, end);

//   pageBooks.forEach((book, index) => {
//     const showDelete = currentUser?.role === "admin";
//     tbody.innerHTML += `
//       <tr>
//         <td>${start + index + 1}</td>
//         <td>${book.title}</td>
//         <td>${book.author}</td>
//         <td>${book.genre || ""}</td>
//         <td>
//           ${book.tags ? book.tags.split(',').map(tag => `<span class="badge bg-secondary me-1 tag-badge" style="cursor:pointer">${tag.trim()}</span>`).join(' ') : ""}
//         </td>
//         <td>
//          ${book.cover
//   ? `<img src="/static/covers/${book.cover}" alt="cover" style="height:50px;cursor:pointer" onclick="showCoverModal('/static/covers/${book.cover}')">`
//   : "—"}
//         </td>
//         <td>
//           <a href="edit.html?id=${book.id}" class="btn btn-sm btn-primary">✏️ Edit</a>
//           ${showDelete ? `<button onclick="deleteBook(${book.id})" class="btn btn-sm btn-danger">🗑️ Delete</button>` : ""}
//         </td>
//       </tr>
//     `;
//   });
// }

function highlightMatch(text, keyword) {
  if (!keyword) return text;
  const regex = new RegExp(`(${keyword})`, "gi");
  return text.replace(regex, '<mark>$1</mark>');
}


function renderBooks() {
  const tbody = document.getElementById("bookTable");
  tbody.innerHTML = "";

  const search = document.getElementById("searchInput").value.toLowerCase();
  const selectedGenre = document.getElementById("genreFilter").value.toLowerCase();

  const filtered = allBooks.filter(b =>
    (b.title.toLowerCase().includes(search) || b.author.toLowerCase().includes(search)) &&
    (selectedGenre === "" || (b.genre && b.genre.toLowerCase() === selectedGenre))
  );

  const sortBy = document.getElementById("sortBy").value;

filtered.sort((a, b) => {
  const valA = sortBy.includes("title") ? a.title.toLowerCase() : a.author.toLowerCase();
  const valB = sortBy.includes("title") ? b.title.toLowerCase() : b.author.toLowerCase();

  if (sortBy.endsWith("asc")) return valA.localeCompare(valB);
  if (sortBy.endsWith("desc")) return valB.localeCompare(valA);
  return 0;
});

  // ✨ Update book count summary
  const summary = document.getElementById("bookSummary");
  summary.textContent = `Showing ${filtered.length} of ${allBooks.length} books`;

  const start = (currentPage - 1) * itemsPerPage;
  const end = start + itemsPerPage;
  const pageBooks = filtered.slice(start, end);

  pageBooks.forEach((book, index) => {
    const showDelete = currentUser?.role === "admin";
    tbody.innerHTML += `
      <tr>
        <td>${start + index + 1}</td>
        <td>${highlightMatch(book.title, search)}</td>
        <td>${highlightMatch(book.author, search)}</td>
        <td>${book.genre || ""}</td>
        <td>
          ${book.tags ? book.tags.split(',').map(tag => `<span class="badge bg-secondary me-1 tag-badge" style="cursor:pointer">${tag.trim()}</span>`).join(' ') : ""}
        </td>
        <td>
          ${book.cover
            ? `<img src="/static/covers/${book.cover}" alt="cover" style="height:50px;cursor:pointer" onclick="showCoverModal('/static/covers/${book.cover}')">`
            : "—"}
        </td>
        <td>
          <a href="edit.html?id=${book.id}" class="btn btn-sm btn-primary">✏️ Edit</a>
          ${showDelete ? `<button onclick="deleteBook(${book.id})" class="btn btn-sm btn-danger">🗑️ Delete</button>` : ""}
        </td>
      </tr>
    `;
  });
}


function renderPagination() {
  const search = document.getElementById("searchInput").value.toLowerCase();
  const total = allBooks.filter(b =>
    b.title.toLowerCase().includes(search) ||
    b.author.toLowerCase().includes(search)
  ).length;

  const pages = Math.ceil(total / itemsPerPage);
  const ul = document.getElementById("pagination");
  ul.innerHTML = "";

  for (let i = 1; i <= pages; i++) {
    ul.innerHTML += `
      <li class="page-item ${i === currentPage ? 'active' : ''}">
        <button class="page-link" onclick="goToPage(${i})">${i}</button>
      </li>
    `;
  }
}

document.getElementById("clearFilters").addEventListener("click", () => {
  document.getElementById("searchInput").value = "";
  document.getElementById("genreFilter").value = "";
  document.getElementById("sortBy").value = "";

  // Optional: hide the "Clear Tag" button if it was shown
  const clearTagBtn = document.getElementById("clearTagBtn");
  if (clearTagBtn) clearTagBtn.style.display = "none";

  currentPage = 1;
  renderBooks();
  renderPagination();
});


// 📌 Clickable Tag Filters
document.addEventListener("click", (e) => {
  if (e.target.classList.contains("tag-badge")) {
    const tag = e.target.textContent.toLowerCase();
    document.getElementById("searchInput").value = tag;
    currentPage = 1;
    renderBooks();
    renderPagination();
  }
});


function goToPage(page) {
  currentPage = page;
  renderBooks();
  renderPagination();
}

function deleteBook(id) {
  bookToDelete = id;
  const modal = new bootstrap.Modal(document.getElementById("deleteModal"));
  modal.show();
}

document.getElementById("confirmDeleteBtn").addEventListener("click", () => {
  if (!bookToDelete) return;
  fetchWithRefresh(`/books/${bookToDelete}`, {
    method: "DELETE",
    headers: getAuthHeaders()
  })
    .then(res => {
      if (!res.ok) throw new Error("Unauthorized");
      fetchBooks();
    })
    .catch(() => {
      alert("❌ Session expired. Please login again.");
      logout();
    });
});


function showCoverModal(src) {
  document.getElementById("modalCoverImg").src = src;
  new bootstrap.Modal(document.getElementById("coverModal")).show();
}


function populateGenreFilter() {
  const genreSelect = document.getElementById("genreFilter");
  const genres = [...new Set(allBooks.map(b => b.genre).filter(Boolean))];
  genreSelect.innerHTML = `<option value="">🎭 All Genres</option>`;
  genres.forEach(g => {
    genreSelect.innerHTML += `<option value="${g}">${g}</option>`;
  });

  genreSelect.addEventListener("change", () => {
    currentPage = 1;
    renderBooks();
    renderPagination();
  });
}


function exportBooks(format) {
  const data = allBooks;
  if (format === "json") {
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "books.json";
    a.click();
  } else if (format === "csv") {
    const rows = [["ID", "Title", "Author"], ...data.map(b => [b.id, b.title, b.author])];
    const csv = rows.map(r => r.join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "books.csv";
    a.click();
  }
}

function toggleDarkMode() {
  const body = document.body;
  const toggleBtn = document.getElementById("darkModeToggle");
  body.classList.toggle("dark-mode");
  const isDark = body.classList.contains("dark-mode");
  localStorage.setItem("darkMode", isDark);
  toggleBtn.innerHTML = isDark ? "🌞" : "🌙";
}

function logout() {
  fetch("/logout", { method: "POST", credentials: "include" });
  localStorage.removeItem("access_token");
  window.location.href = "login.html";
}

document.getElementById("searchInput").addEventListener("input", () => {
  currentPage = 1;
  renderBooks();
  renderPagination();
});

applyDarkModePreference();

if (!getAccessToken()) {
  alert("⚠️ Please login to continue.");
  window.location.href = "login.html";
} else {
  fetchCurrentUser();
}

document.getElementById("sortBy").addEventListener("change", () => {
  currentPage = 1;
  renderBooks();
  renderPagination();
});

window.addEventListener("DOMContentLoaded", () => {
  const isDark = localStorage.getItem("darkMode") === "true";
  if (isDark) {
    document.body.classList.add("dark-mode");
  }
  const toggleBtn = document.getElementById("darkModeToggle");
  if (toggleBtn) {
    toggleBtn.innerHTML = isDark ? "🌞" : "🌙";
  }
  if (!getAccessToken()) {
    alert("⚠️ Please login to continue.");
    window.location.href = "login.html";
  } else {
    fetchCurrentUser();
  }
});

document.addEventListener("click", (e) => {
  if (e.target.classList.contains("tag-badge")) {
    const tag = e.target.textContent.trim().toLowerCase();
    document.getElementById("searchInput").value = tag;
    currentPage = 1;
    renderBooks();
    renderPagination();
  }
});
