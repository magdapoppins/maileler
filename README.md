## 🍋 mailerer - a small calendar management util

Messy as hell but it works. Uses [bubbletea](https://github.com/charmbracelet/bubbletea)!

### Available functionality
- Walk through your current week calendar
- Find available 1h slots for meetings
- Send an email with available meeting slots

### To use 
1. Create a credentials.json file in gc, add it to the repository root as `credentials.json`- it needs access to calendar and email.
2. Run
```
go test ./...
go run .
```