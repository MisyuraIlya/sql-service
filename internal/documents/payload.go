package documents

import "time"

type AllProductsDto struct {
	UserExtId string `json:"userExtId"`
}

type CartessetDto struct {
	CardCode string `json:"cardCode"`
	DateFrom string `json:"dateFrom"`
	DateTo   string `json:"dateTo"`
}

type HovotDto struct {
	CardCode string `json:"cardCode"`
}

type Cartesset struct {
	DocDate        *time.Time `json:"docDate"`
	DueDate        *time.Time `json:"dueDate"`
	DocType        string     `json:"docType"`
	DocNum         *string    `json:"docNum"`
	NumAtCard      *string    `json:"numAtCard"`
	ConfNum        *string    `json:"confNum"`
	Hova           float64    `json:"hova"`
	Zchut          float64    `json:"zchut"`
	RunningBalance float64    `json:"runningBalance"`
}

type Hovot struct {
	DueDate     *time.Time `json:"dueDate"`
	DocDate     *time.Time `json:"docDate"`
	DocType     string     `json:"docType"`
	DocNum      *string    `json:"docNum"`
	NumAtCard   *string    `json:"numAtCard"`
	ConfNum     *string    `json:"confNum"`
	Amount      float64    `json:"amount"`
	RunningOpen float64    `json:"runningOpen"`
}

type OpenProducts struct {
	ItemCode      string `json:"itemCode"`
	TotalOpenQty  int    `json:"totalOpenQty"`
	DocNumbers    string `json:"docNumbers"`
	NumAtCard     string `json:"numAtCard"`
	OrderDocDates string `json:"orderDocDates"`
	LineDocDates  string `json:"lineDocDates"`
	AvailStatuses string `json:"availStatuses"`
	FreeTexts     string `json:"freeTexts"`
}
