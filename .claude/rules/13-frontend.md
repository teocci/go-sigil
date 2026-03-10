# Frontend (Vanilla JS + CSS)

## Core Principle

**No magic, no black boxes.** Vanilla JS and CSS unless there's a compelling reason otherwise.

## Directory Structure

```
web/
├── static/
│   ├── css/
│   │   ├── variables.css    # CSS custom properties
│   │   ├── base.css         # Reset, typography
│   │   └── components/
│   │       ├── button.css
│   │       └── card.css
│   ├── js/
│   │   ├── app.js           # Entry point
│   │   ├── api.js           # API client
│   │   └── components/
│   │       ├── viewer.js
│   │       └── toolbar.js
│   └── icons/               # Custom SVG icons
│       ├── search.svg
│       └── user.svg
└── templates/
    ├── layouts/
    │   └── base.html
    └── pages/
        ├── index.html
        └── dashboard.html
```

## JavaScript Conventions

### $ Prefix for DOM Elements

```javascript
// DOM elements - prefixed with $
const $viewer = document.getElementById('viewer')
const $sidebar = document.querySelector('.sidebar')

// Components/data - no prefix
const viewer = new ViewerComponent($viewer)
const config = { theme: 'dark' }
```

### Module Pattern

```javascript
// components/viewer.js
export default class ViewerComponent {
    static TAG = 'viewer'

    /** @type {HTMLElement} */
    $element

    constructor($element) {
        this.$element = $element
        this.init()
    }

    get isActive() {
        return this.$element.classList.contains('active')
    }

    init() {
        this.bindEvents()
    }

    bindEvents() {
        this.$element.addEventListener('click', e => {
            const $target = e.target.closest('[data-action]')
            if ($target) this.handleAction($target.dataset.action)
        })
    }

    destroy() {
        this.$element = null
    }
}
```

### API Client

```javascript
// api.js
const API_BASE = '/api/v1'

async function request(method, path, data = null) {
    const options = {
        method,
        headers: {
            'Content-Type': 'application/json'
        }
    }

    if (data) {
        options.body = JSON.stringify(data)
    }

    const token = localStorage.getItem('token')
    if (token) {
        options.headers['Authorization'] = `Bearer ${token}`
    }

    const response = await fetch(`${API_BASE}${path}`, options)

    if (!response.ok) {
        const error = await response.json()
        throw new Error(error.message || 'Request failed')
    }

    return response.json()
}

export const api = {
    get: (path) => request('GET', path),
    post: (path, data) => request('POST', path, data),
    put: (path, data) => request('PUT', path, data),
    delete: (path) => request('DELETE', path)
}
```

## CSS Conventions

### CSS Custom Properties

```css
/* variables.css */
:root {
    /* Colors */
    --color-primary: #2563eb;
    --color-primary-hover: #1d4ed8;
    --color-text: #1f2937;
    --color-text-muted: #6b7280;
    --color-background: #ffffff;
    --color-border: #e5e7eb;

    /* Spacing */
    --space-xs: 0.25rem;
    --space-sm: 0.5rem;
    --space-md: 1rem;
    --space-lg: 1.5rem;
    --space-xl: 2rem;

    /* Typography */
    --font-family: system-ui, sans-serif;
    --font-size-sm: 0.875rem;
    --font-size-base: 1rem;
    --font-size-lg: 1.125rem;

    /* Borders */
    --radius-sm: 0.25rem;
    --radius-md: 0.5rem;
    --radius-lg: 0.75rem;

    /* Transitions */
    --transition-fast: 150ms ease;
    --transition-base: 200ms ease;
}
```

### Component CSS

```css
/* components/card.css */
.card {
    display: flex;
    gap: var(--space-md);
    padding: var(--space-lg);
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    transition: box-shadow var(--transition-fast);
}

.card:hover {
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.card-title {
    font-size: var(--font-size-lg);
    font-weight: 600;
    color: var(--color-text);
}
```

## No Emojis - Custom SVG Icons

```html
<!-- Bad -->
<button>🔍 Search</button>

<!-- Good -->
<button>
    <svg class="icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8"/>
        <path d="M21 21l-4.35-4.35"/>
    </svg>
    <span>Search</span>
</button>
```

## Serving Static Files (Fiber)

```go
// internal/server/static.go
package server

import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/filesystem"
    "net/http"
)

func SetupStatic(app *fiber.App) {
    // Serve static files
    app.Use("/static", filesystem.New(filesystem.Config{
        Root:   http.Dir("./web/static"),
        Browse: false,
    }))

    // Or embed in binary
    // app.Use("/static", filesystem.New(filesystem.Config{
    //     Root:       http.FS(embeddedFiles),
    //     PathPrefix: "web/static",
    // }))
}
```

## Template Rendering (Fiber)

```go
// internal/server/templates.go
package server

import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/template/html/v2"
)

func SetupTemplates(app *fiber.App) {
    engine := html.New("./web/templates", ".html")

    // Enable reload in development
    if config.IsDevelopment() {
        engine.Reload(true)
    }

    app.Settings.Views = engine
}

// Handler
func (h *Handler) Dashboard(c *fiber.Ctx) error {
    return c.Render("pages/dashboard", fiber.Map{
        "Title": "Dashboard",
        "User":  c.Locals("user"),
    }, "layouts/base")
}
```

## When to Use SASS/LESS

Only if:
- Complex design system with deep theming
- Legacy codebase already using it
- Team consensus after evaluating trade-offs

Modern CSS covers most needs with:
- CSS Custom Properties (variables)
- CSS Nesting (now native)
- CSS Grid and Flexbox
- CSS Container Queries
