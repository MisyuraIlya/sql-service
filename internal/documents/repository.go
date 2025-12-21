package documents

import (
	"context"
	"database/sql"

	"sql-service/pkg/db"
)

type DocumentRrepository struct {
	Db *db.Db
}

func NewDocumentRepository(db *db.Db) *DocumentRrepository {
	return &DocumentRrepository{Db: db}
}

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

func (r *DocumentRrepository) GetOpenProducts(dto *AllProductsDto) ([]OpenProducts, error) {
	const query = `
		SELECT
			r.ItemCode,
			r.OpenQty AS TotalOpenQty,
			CONVERT(varchar(20), o.DocNum)       AS DocNumbers,
			ISNULL(o.NumAtCard, '')              AS NumAtCard,
			CONVERT(varchar(10), o.DocDate, 23)  AS OrderDocDates,
			CONVERT(varchar(10), r.DocDate, 23)  AS LineDocDates,
			ISNULL(r.U_AvailStat, '')            AS AvailStatuses,
			ISNULL(r.FreeTxt, '')                AS FreeTexts
		FROM RDR1 r
		JOIN ORDR o ON o.DocEntry = r.DocEntry
		WHERE r.LineStatus = 'O'
		AND o.CANCELED = 'N'
		AND o.CardCode = @cardCode
		ORDER BY r.ItemCode, o.DocNum;
	`

	ctx := context.Background()

	rows, err := r.Db.QueryContext(
		ctx,
		query,
		sql.Named("cardCode", dto.UserExtId),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]OpenProducts, 0, 64)
	for rows.Next() {
		var (
			itemCode      string
			totalOpenQty  float64
			docNumbers    string
			numAtCard     string
			orderDocDates string
			lineDocDates  string
			availStatuses string
			freeTexts     string
		)

		if err := rows.Scan(
			&itemCode,
			&totalOpenQty,
			&docNumbers,
			&numAtCard,
			&orderDocDates,
			&lineDocDates,
			&availStatuses,
			&freeTexts,
		); err != nil {
			return nil, err
		}

		out = append(out, OpenProducts{
			ItemCode:      itemCode,
			TotalOpenQty:  int(totalOpenQty),
			DocNumbers:    docNumbers,
			NumAtCard:     numAtCard,
			OrderDocDates: orderDocDates,
			LineDocDates:  lineDocDates,
			AvailStatuses: availStatuses,
			FreeTexts:     freeTexts,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
