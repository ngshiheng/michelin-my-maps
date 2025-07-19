package backfill

const (
	// Award data extraction XPaths for backfill
	awardDistinctionXPath1 = "//ul[contains(@class,'restaurant-details__classification--list')]//li"
	awardDistinctionXPath2 = "//div[contains(@class,'restaurant__classification')]//p[contains(@class,'flex-fill')]"

	// Price extraction XPaths
	awardPriceXPath1    = "//li[contains(@class,'restaurant-details__heading-price')]"
	awardPriceXPath2    = "//li[contains(.,'span') and (contains(.,'mg-price') or contains(.,'mg-euro-circle'))]"
	awardPriceXPath3    = "//div[contains(@class,'data-sheet__block--text')][2]"
	awardPriceSpanXPath = ".//span"

	// Published date extraction XPaths
	awardDateXPath1    = "//div[contains(@class,'restaurant-details__heading--label-title')]"
	awardDateXPath2    = "//div[contains(@class,'label-text')]"
	awardDateMetaXPath = "//meta[@name='description']"

	// Script tags for dLayer and JSON-LD
	awardScriptXPath = "//script"
	awardJSONLDXPath = "//script[@type='application/ld+json']"
)
