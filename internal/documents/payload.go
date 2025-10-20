package documents

import "time"

type CartessetDto struct {
	CardCode string `json:cardCode`
	DateFrom string `json:dateFrom`
	DateTo   string `json:dateTo`
}

type Cartesset struct {
	CreateDate time.Time `json:"createDate"`
	DueDate    time.Time `json:"dueDate"`
	DocType    string    `json:"docType"`
	BaseRef    string    `json:"baseRef"`
	Ref1       string    `json:"ref1"`
	Ref2       string    `json:"ref2"`
	TransId    int       `json:"transId"`
	ShortName  string    `json:"shortName"`
	Memo       string    `json:"memo"`
	Debit      float64   `json:"debit"`
	Credit     float64   `json:"credit"`
	CardCode   string    `json:"cardCode"`
	CardName   string    `json:"cardName"`
}

type OpenProducts struct {
	ItemCode     string `json:"itemCode"`
	TotalOpenQty int    `json:totalOpenQty`
	DocNumbers   string `json:DocNumber`
}
