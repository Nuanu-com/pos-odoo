# How to use

### Create a new client
```go
import (
	"net/http"
	"net/http/cookiejar"
	odoo "github.com/Nuanu-com/pos-odoo"
)

func main() {
	jar, _ := cookiejar.New(nil)

	auth := odoo.AddAuthentication(func() (string, string, string) {
		return config.OdooUsername, config.OdooPassword, config.OdooDB
	})

	client := odoo.NewOdooClient(&http.Client{Jar: jar}, config.OdooBaseURL, auth)
}
```

### Authenticate with an API key (recommended)

Generate a key in Odoo under **Preferences → Account Security → New API Key** for the integration user, then pair it with that user's login. Calls go through `/jsonrpc` (`execute_kw`), so no cookie jar is needed and there's no session to expire.

```go
auth := odoo.AddAPIKeyAuthentication(func() (string, string, string) {
	return config.OdooLogin, config.OdooAPIKey, config.OdooDB
})

client := odoo.NewOdooClient(&http.Client{}, config.OdooBaseURL, auth)
```

Note: unlike session auth, the server doesn't inject the user's context (lang, tz, `allowed_company_ids`) — pass it explicitly where it matters, e.g. `SearchReadAccountTaxWithOptions{Context: ...}`.

For more detailed example, check
- [Purchase Create](https://github.com/Nuanu-com/kiosk-backend/blob/main/app/tasks/purchase_sync_odoo_task.go)
- [Ticket Sync](https://github.com/Nuanu-com/kiosk-backend/blob/main/app/tasks/ticket_sync_odoo_task.go)
- [Account Tax](https://github.com/Nuanu-com/kiosk-backend/blob/main/app/tasks/account_tax_sync_task.go)
- [Session](https://github.com/Nuanu-com/kiosk-backend/blob/main/app/tasks/pos_session_task.go)
