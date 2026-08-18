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

For more detailed example, check
- [Purchase Create](https://github.com/Nuanu-com/kiosk-backend/blob/main/app/tasks/purchase_sync_odoo_task.go)
- [Ticket Sync](https://github.com/Nuanu-com/kiosk-backend/blob/main/app/tasks/ticket_sync_odoo_task.go)
- [Account Tax](https://github.com/Nuanu-com/kiosk-backend/blob/main/app/tasks/account_tax_sync_task.go)
- [Session](https://github.com/Nuanu-com/kiosk-backend/blob/main/app/tasks/pos_session_task.go)
