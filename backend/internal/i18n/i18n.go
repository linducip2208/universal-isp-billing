package i18n

import "encoding/json"

// Dict holds translation strings. Frontend mirrors these keys; backend uses
// them for notification templates. No hardcoded UI strings elsewhere.
var en = map[string]string{
	"nav.dashboard": "Dashboard", "nav.noc": "NOC", "nav.customers": "Customers",
	"nav.billing": "Billing", "nav.network": "Network", "nav.services": "Services",
	"nav.ftth": "FTTH", "nav.monitoring": "Monitoring", "nav.automation": "Automation",
	"nav.reports": "Reports", "nav.system": "System",
	"common.search": "Search", "common.save": "Save", "common.cancel": "Cancel",
	"common.status": "Status", "common.actions": "Actions",
	"noc.title":        "Network Operations Center",
	"billing.invoices": "Invoices", "billing.payments": "Payments",
	"auth.login": "Sign in",
}

var id = map[string]string{
	"nav.dashboard": "Dasbor", "nav.noc": "NOC", "nav.customers": "Pelanggan",
	"nav.billing": "Penagihan", "nav.network": "Jaringan", "nav.services": "Layanan",
	"nav.ftth": "FTTH", "nav.monitoring": "Pemantauan", "nav.automation": "Otomasi",
	"nav.reports": "Laporan", "nav.system": "Sistem",
	"common.search": "Cari", "common.save": "Simpan", "common.cancel": "Batal",
	"common.status": "Status", "common.actions": "Aksi",
	"noc.title":        "Pusat Operasi Jaringan",
	"billing.invoices": "Faktur", "billing.payments": "Pembayaran",
	"auth.login": "Masuk",
}

func T(lang, key string) string {
	if lang == "id" {
		if v, ok := id[key]; ok {
			return v
		}
	}
	if v, ok := en[key]; ok {
		return v
	}
	return key
}

func Marshal(lang string) string {
	m := en
	if lang == "id" {
		m = id
	}
	b, _ := json.Marshal(m)
	return string(b)
}
