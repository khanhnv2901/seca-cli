# HTML Template Approaches

This document explains the different approaches for managing HTML templates in the report generation feature.

## Current Implementation: External Template with `embed`

### Structure
```
cmd/
├── report.go                    # Main report generation code
└── templates/
    └── report.html              # HTML template file
```

### Advantages
✅ **Clean separation of concerns** - HTML/CSS in separate file
✅ **Better IDE support** - Syntax highlighting for HTML/CSS
✅ **Easier to maintain** - Edit HTML without dealing with Go string escaping
✅ **No runtime dependencies** - Template embedded in binary at compile time
✅ **Template reusability** - Can create multiple templates easily
✅ **Type-safe** - Using Go's `html/template` with struct binding

### How It Works

1. **Template file** ([cmd/templates/report.html](cmd/templates/report.html))
   - Standard HTML file with Go template syntax
   - Uses `{{.FieldName}}` for data binding
   - Supports conditionals: `{{if .Field}}...{{end}}`
   - Supports loops: `{{range .Items}}...{{end}}`

2. **Embed directive** ([report.go:16-17](cmd/report.go#L16-L17))
   ```go
   //go:embed templates/report.html
   var htmlTemplate string
   ```
   - Reads template file at compile time
   - Embedded in the final binary
   - No external file needed at runtime

3. **Template data structure** ([report.go:174-186](cmd/report.go#L174-L186))
   ```go
   type TemplateData struct {
       Metadata     RunMetadata
       Results      []CheckResult
       GeneratedAt  string
       // ... other fields
   }
   ```

4. **Template execution** ([report.go:188-232](cmd/report.go#L188-L232))
   ```go
   tmpl, err := template.New("report").Parse(htmlTemplate)
   err = tmpl.Execute(&buf, data)
   ```

### Example: Adding a New Field

**Step 1:** Add field to TemplateData struct
```go
type TemplateData struct {
    // ... existing fields
    CustomField string
}
```

**Step 2:** Populate in generateHTMLReport
```go
data := TemplateData{
    // ... existing data
    CustomField: "my value",
}
```

**Step 3:** Use in template
```html
<div>{{.CustomField}}</div>
```

## Alternative: Inline Template (Previous Implementation)

### Structure
```go
func generateHTMLReport(output *RunOutput) (string, error) {
    var sb strings.Builder
    sb.WriteString("<!DOCTYPE html>\n")
    sb.WriteString("<html>...")
    // ... 200+ lines of string concatenation
}
```

### When to Use This Approach
- Very simple, small HTML snippets
- One-time use templates
- No need for template logic (conditionals, loops)

### Disadvantages
❌ HTML mixed with Go code
❌ Difficult to read and maintain
❌ Poor IDE support for HTML/CSS
❌ String escaping issues
❌ Hard to preview template

## Alternative: Multiple Template Files

For complex applications with multiple report types:

```
cmd/templates/
├── report.html           # Standard report
├── summary.html          # Summary report
└── detailed.html         # Detailed report
```

```go
//go:embed templates/*.html
var templates embed.FS

// Load specific template
tmpl, err := template.ParseFS(templates, "templates/report.html")
```

## Best Practices

### 1. Use External Templates When:
- HTML is more than ~20 lines
- Need conditionals or loops
- Want to version control HTML separately
- Multiple people editing (Go dev vs. designer)

### 2. Use Inline When:
- Very simple HTML snippets (< 10 lines)
- One-time use
- No template logic needed

### 3. Template Security
- Always use `html/template` (not `text/template`) for HTML
- Auto-escapes HTML, preventing XSS attacks
- Example: `<script>` tags are automatically escaped

### 4. Template Organization
```
templates/
├── report.html          # Main templates
├── email.html
└── partials/            # Reusable components
    ├── header.html
    └── footer.html
```

## Migration Guide

### From Inline to External Template

**Before:**
```go
func generateHTMLReport(output *RunOutput) (string, error) {
    var sb strings.Builder
    sb.WriteString("<h1>Title</h1>")
    sb.WriteString(fmt.Sprintf("<p>%s</p>", output.Data))
    return sb.String(), nil
}
```

**After:**

1. Create template file:
```html
<!-- templates/mytemplate.html -->
<h1>Title</h1>
<p>{{.Data}}</p>
```

2. Embed and use:
```go
//go:embed templates/mytemplate.html
var myTemplate string

func generateHTMLReport(output *RunOutput) (string, error) {
    tmpl, err := template.New("report").Parse(myTemplate)
    if err != nil {
        return "", err
    }

    var buf strings.Builder
    err = tmpl.Execute(&buf, output)
    return buf.String(), err
}
```

## Testing Templates

Tests work the same way regardless of approach:

```go
func TestGenerateHTMLReport(t *testing.T) {
    output := &RunOutput{...}

    report, err := generateHTMLReport(output)
    if err != nil {
        t.Fatalf("Failed: %v", err)
    }

    if !strings.Contains(report, "expected content") {
        t.Error("Template missing expected content")
    }
}
```

## Performance

- **Compile time:** Template embedded, no performance difference
- **Runtime:** Template parsing cached if needed
- **Binary size:** Minimal increase (few KB for typical templates)

## Conclusion

The current implementation using external templates with `embed` provides the best balance of:
- Code organization
- Maintainability
- Developer experience
- Deployment simplicity (single binary)

Choose inline templates only for very simple cases where the overhead of a separate file isn't justified.
