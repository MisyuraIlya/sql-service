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
;WITH Lines AS
(
    SELECT
        T0.RefDate AS DocumentDate,
        COALESCE(OINV.DocDueDate, ORIN.DocDueDate, CAST(T0.DueDate AS date)) AS DocDueDate,
        T0.TransId,
        T1.Line_ID,

        CASE T0.TransType
            WHEN 13 THEN N'חשבונית'
            WHEN 14 THEN N'חשבונית זיכוי'
            WHEN 24 THEN N'קבלה'
            WHEN 30 THEN N'תנועת יומן'
            ELSE N'תנועה'
        END AS DocumentType,

        COALESCE(
            CAST(OINV.DocNum AS nvarchar(30)),
            CAST(ORIN.DocNum AS nvarchar(30)),
            CAST(ORCT.DocNum AS nvarchar(30)),
            NULLIF(T0.BaseRef, ''),
            CAST(T0.TransId AS nvarchar(30))
        ) AS DocumentNumber,

        COALESCE(OINV.NumAtCard, ORIN.NumAtCard) AS NumAtCard,

        RIGHT(
            NULLIF(LTRIM(RTRIM(COALESCE(OINV.U_INS_TDI_CONFNUM, ORIN.U_INS_TDI_CONFNUM))), ''),
            9
        ) AS ConfNum,

        CAST(ISNULL(T1.Debit, 0)  AS decimal(19,2)) AS Debit,
        CAST(ISNULL(T1.Credit, 0) AS decimal(19,2)) AS Credit,

        CAST(ISNULL(T1.Debit, 0) - ISNULL(T1.Credit, 0) AS decimal(19,2)) AS NetAmount,

        CAST(
            CASE
                WHEN T0.TransType IN (13,14) THEN (ISNULL(T1.Debit,0) - ISNULL(T1.Credit,0))
                WHEN T0.TransType = 30 AND (ISNULL(T1.Debit,0) - ISNULL(T1.Credit,0)) > 0
                    THEN (ISNULL(T1.Debit,0) - ISNULL(T1.Credit,0))
                ELSE 0
            END
        AS decimal(19,2)) AS Hova,

        CAST(
            CASE
                WHEN T0.TransType = 24 THEN (ISNULL(T1.Credit,0) - ISNULL(T1.Debit,0))
                WHEN T0.TransType = 30 AND (ISNULL(T1.Credit,0) - ISNULL(T1.Debit,0)) > 0
                    THEN (ISNULL(T1.Credit,0) - ISNULL(T1.Debit,0))
                ELSE 0
            END
        AS decimal(19,2)) AS Zchut
    FROM OJDT T0
    INNER JOIN JDT1 T1 ON T1.TransId = T0.TransId
    LEFT JOIN OINV ON OINV.TransId = T0.TransId AND T0.TransType = 13
    LEFT JOIN ORIN ON ORIN.TransId = T0.TransId AND T0.TransType = 14
    LEFT JOIN ORCT ON ORCT.TransId = T0.TransId AND T0.TransType = 24
    WHERE
        T1.ShortName = @cardCode
        AND T0.RefDate >= @fromDate
        AND T0.RefDate <= @toDate
        AND (ISNULL(T1.Debit,0) <> 0 OR ISNULL(T1.Credit,0) <> 0)
),
Opening AS
(
    SELECT
        CAST(SUM(ISNULL(T1.Debit,0) - ISNULL(T1.Credit,0)) AS decimal(19,2)) AS OpeningBalance
    FROM OJDT T0
    INNER JOIN JDT1 T1 ON T1.TransId = T0.TransId
    WHERE
        T1.ShortName = @cardCode
        AND T0.RefDate < @fromDate
        AND (ISNULL(T1.Debit,0) <> 0 OR ISNULL(T1.Credit,0) <> 0)
),
FinalData AS
(
    SELECT
        0 AS SortRow,
        CAST(NULL AS date) AS DocDate,
        CAST(NULL AS date) AS DueDate,
        N'יתרת פתיחה' AS DocType,
        CAST(NULL AS nvarchar(30)) AS DocNum,
        CAST(NULL AS nvarchar(100)) AS NumAtCard,
        CAST(NULL AS nvarchar(9)) AS ConfNum,
        CAST(0 AS decimal(19,2)) AS Hova,
        CAST(0 AS decimal(19,2)) AS Zchut,
        CAST(ISNULL(O.OpeningBalance,0) AS decimal(19,2)) AS RunningBalance,
        CAST(NULL AS int) AS TransId,
        CAST(NULL AS int) AS LineId
    FROM Opening O

    UNION ALL

    SELECT
        1,
        L.DocumentDate,
        L.DocDueDate,
        L.DocumentType,
        L.DocumentNumber,
        L.NumAtCard,
        L.ConfNum,
        L.Hova,
        L.Zchut,
        CAST(
            ISNULL((SELECT OpeningBalance FROM Opening),0)
            + SUM(L.NetAmount) OVER (ORDER BY L.DocumentDate, L.TransId, L.Line_ID ROWS UNBOUNDED PRECEDING)
        AS decimal(19,2)),
        L.TransId,
        L.Line_ID
    FROM Lines L
)

SELECT
    DocDate,
    DueDate,
    DocType,
    DocNum,
    NumAtCard,
    ConfNum,
    Hova,
    Zchut,
    RunningBalance
FROM FinalData
ORDER BY
    SortRow,
    DocDate,
    DueDate,
    TransId,
    LineId;
    `

	ctx := context.Background()

	rows, err := r.Db.QueryContext(ctx, query,
		sql.Named("cardCode", dto.CardCode),
		sql.Named("fromDate", dto.DateFrom),
		sql.Named("toDate", dto.DateTo),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Cartesset
	for rows.Next() {
		var c Cartesset
		if err := rows.Scan(
			&c.DocDate,
			&c.DueDate,
			&c.DocType,
			&c.DocNum,
			&c.NumAtCard,
			&c.ConfNum,
			&c.Hova,
			&c.Zchut,
			&c.RunningBalance,
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

func (r *DocumentRrepository) GetHovot(dto *HovotDto) ([]Hovot, error) {
	const query = `
;WITH J AS
(
    SELECT
        T0.RefDate AS DocDate,
        CAST(T0.DueDate AS date) AS DueDate,
        T0.TransId,
        T1.Line_ID,
        T0.TransType,
        CAST(ISNULL(T1.BalDueDeb,0) - ISNULL(T1.BalDueCred,0) AS decimal(19,2)) AS Amount,
        CASE T0.TransType
            WHEN 13 THEN N'חשבונית'
            WHEN 14 THEN N'חשבונית זיכוי'
            WHEN 24 THEN N'קבלה'
            WHEN 30 THEN N'תנועת יומן'
            ELSE N'תנועה'
        END AS DocType
    FROM OJDT T0
    INNER JOIN JDT1 T1 ON T1.TransId = T0.TransId
    WHERE
        T1.ShortName = @cardCode
        AND (ISNULL(T1.BalDueDeb,0) <> 0 OR ISNULL(T1.BalDueCred,0) <> 0)
),
DataWithDoc AS
(
    SELECT
        J.DocDate,
        J.DueDate,
        J.DocType,
        COALESCE(
            CASE WHEN J.TransType = 13 THEN CAST(I.DocNum AS nvarchar(30)) END,
            CASE WHEN J.TransType = 14 THEN CAST(C.DocNum AS nvarchar(30)) END,
            CAST(J.TransId AS nvarchar(30))
        ) AS DocNum,
        COALESCE(
            CASE WHEN J.TransType = 13 THEN I.NumAtCard END,
            CASE WHEN J.TransType = 14 THEN C.NumAtCard END
        ) AS NumAtCard,
        RIGHT(
            NULLIF(LTRIM(RTRIM(
                COALESCE(
                    CASE WHEN J.TransType = 13 THEN I.U_INS_TDI_CONFNUM END,
                    CASE WHEN J.TransType = 14 THEN C.U_INS_TDI_CONFNUM END
                )
            )), ''),
            9
        ) AS ConfNum,
        J.Amount,
        J.TransId,
        J.Line_ID
    FROM J
    LEFT JOIN OINV I ON I.TransId = J.TransId AND J.TransType = 13
    LEFT JOIN ORIN C ON C.TransId = J.TransId AND J.TransType = 14
),
FinalData AS
(
    SELECT
        D.DocDate,
        D.DueDate,
        D.DocType,
        D.DocNum,
        D.NumAtCard,
        D.ConfNum,
        D.Amount,
        CAST(
            SUM(D.Amount) OVER (ORDER BY D.DueDate, D.DocDate, D.TransId, D.Line_ID ROWS UNBOUNDED PRECEDING)
        AS decimal(19,2)) AS RunningOpen,
        D.TransId,
        D.Line_ID AS LineId
    FROM DataWithDoc D
)
SELECT
    DueDate,
    DocDate,
    DocType,
    DocNum,
    NumAtCard,
    ConfNum,
    Amount,
    RunningOpen
FROM FinalData
ORDER BY DueDate, DocDate, TransId, LineId;
    `

	ctx := context.Background()

	rows, err := r.Db.QueryContext(ctx, query,
		sql.Named("cardCode", dto.CardCode),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Hovot
	for rows.Next() {
		var h Hovot
		if err := rows.Scan(
			&h.DueDate,
			&h.DocDate,
			&h.DocType,
			&h.DocNum,
			&h.NumAtCard,
			&h.ConfNum,
			&h.Amount,
			&h.RunningOpen,
		); err != nil {
			return nil, err
		}
		out = append(out, h)
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
