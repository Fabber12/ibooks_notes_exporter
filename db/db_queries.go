package db

const GetAllBooksDbQueryConstant = `
        select
                ZBKLIBRARYASSET.ZASSETID,
                ZBKLIBRARYASSET.ZTITLE,
                ZBKLIBRARYASSET.ZAUTHOR,
                count(a.ZAEANNOTATION.Z_PK)
        from ZBKLIBRARYASSET left join a.ZAEANNOTATION
                on a.ZAEANNOTATION.ZANNOTATIONASSETID = ZBKLIBRARYASSET.ZASSETID
        WHERE a.ZAEANNOTATION.ZANNOTATIONSELECTEDTEXT NOT NULL
        GROUP BY ZBKLIBRARYASSET.ZASSETID;
`

const GetBookDataById = `
        select
                ZBKLIBRARYASSET.ZTITLE,
                ZBKLIBRARYASSET.ZAUTHOR
        from ZBKLIBRARYASSET
        where ZBKLIBRARYASSET.ZASSETID=$1
`

const GetNotesHighlightsById = `
        SELECT
                ZANNOTATIONSELECTEDTEXT,
                ZANNOTATIONNOTE,
                ZANNOTATIONSTYLE,
                ZANNOTATIONISUNDERLINE
        FROM
                ZAEANNOTATION
        WHERE
                ZANNOTATIONASSETID = $1
                AND ZANNOTATIONSELECTEDTEXT IS NOT NULL
        ORDER BY
                ZPLLOCATIONRANGESTART ASC,
                ZANNOTATIONCREATIONDATE ASC
        LIMIT 1000000 OFFSET $2
`
