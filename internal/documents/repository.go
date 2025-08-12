package documents

import (
	"context"
	"database/sql"
	"time"

	"sql-service/pkg/db"
)

// DTO for passing filter parameters into GetCartesset

// Cartesset represents one row of the result set
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

// Repository holds your DB connection
type DocumentRrepository struct {
	Db *db.Db
}

func NewDocumentRepository(db *db.Db) *DocumentRrepository {
	return &DocumentRrepository{Db: db}
}

// GetCartesset fetches all matching documents for the given card and date range.
func (r *DocumentRrepository) GetCartesset(dto *CartessetDto) ([]Cartesset, error) {
	const query = `
    SELECT
        T0.CreateDate,
        T0.DueDate,
        CASE T0.[TransType]
            WHEN '24' THEN N'קבלות'
            WHEN '13' THEN N'חשבונית'
            WHEN '14' THEN N'חשבונית זיכוי'
        END AS DocType,
        T0.BaseRef,
        T0.Ref1,
        T0.Ref2,
        T0.TransId,
        T1.ShortName,
        T0.Memo,
        T1.Debit,
        T1.Credit,
        T3.CardCode,
        T3.CardName
    FROM dbo.OJDT AS T0
    INNER JOIN dbo.JDT1 AS T1 ON T0.TransId = T1.TransId
    INNER JOIN dbo.OACT AS T2 ON T1.Account = T2.AcctCode
    INNER JOIN dbo.OCRD AS T3 ON T1.ShortName = T3.CardCode
    WHERE
        T3.CardType = 'C'
        AND T0.TransType IN ('13', '14', '24')
        AND T3.CardCode = @cardCode
        AND T0.CreateDate BETWEEN @from AND @to;
    `

	ctx := context.Background()

	rows, err := r.Db.QueryContext(ctx, query,
		sql.Named("cardCode", dto.CardCode),
		sql.Named("from", dto.DateFrom),
		sql.Named("to", dto.DateTo),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Cartesset
	for rows.Next() {
		var c Cartesset
		if err := rows.Scan(
			&c.CreateDate,
			&c.DueDate,
			&c.DocType,
			&c.BaseRef,
			&c.Ref1,
			&c.Ref2,
			&c.TransId,
			&c.ShortName,
			&c.Memo,
			&c.Debit,
			&c.Credit,
			&c.CardCode,
			&c.CardName,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
