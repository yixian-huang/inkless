package builtinthemes

import _ "embed"

//go:embed pages.json
var PagesJSON []byte

// EditorialFirmSeedsJSON is the default unified-page section configs for
// editorial-firm (home/about/services/contact). Applied on activate when
// published sections are empty — see theme page seed helper.
//
//go:embed editorial_firm_seeds.json
var EditorialFirmSeedsJSON []byte
