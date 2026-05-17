package ingestion

import (
	"bufio"
	"log"
	"net/http"
	"strconv"
	"strings"

	"MFBasketBacktester/internal/db"
)

const AMFIURL = "https://portal.amfiindia.com/spages/NAVAll.txt"

func FetchAndStoreNAV() {
	resp, err := http.Get(AMFIURL)
	if err != nil {
		log.Println("Failed to fetch AMFI data:", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.Split(line, ";")

		// A valid NAV line has 6 fields: scheme code, two ISINs, scheme
		// name, NAV, and date. Section headers and blank lines have fewer.
		if len(parts) < 6 {
			continue
		}

		schemeCode, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}

		schemeName := parts[3]

		navValue, err := strconv.ParseFloat(parts[4], 64)
		if err != nil {
			continue
		}

		// Date is the final field; TrimSpace also drops any trailing CR.
		date := strings.TrimSpace(parts[5])
		if date == "" {
			continue
		}

		var fundID int

		err = db.DB.QueryRow(`
			INSERT INTO funds (scheme_code, scheme_name)
			VALUES ($1, $2)
			ON CONFLICT (scheme_code)
			DO UPDATE SET scheme_name = EXCLUDED.scheme_name
			RETURNING id
		`, schemeCode, schemeName).Scan(&fundID)

		if err != nil {
			log.Println("Fund insert failed:", err)
			continue
		}

		_, err = db.DB.Exec(`
			INSERT INTO nav (fund_id, nav, date)
			VALUES ($1, $2, TO_DATE($3, 'DD-Mon-YYYY'))
			ON CONFLICT (fund_id, date)
			DO NOTHING
		`, fundID, navValue, date)

		if err != nil {
			log.Println("NAV insert failed:", err)
			continue
		}
	}

	log.Println("AMFI NAV ingestion complete")
}
