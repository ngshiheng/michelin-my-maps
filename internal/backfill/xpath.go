package backfill

const (
	// Award data extraction XPaths for backfill
	awardDistinctionXPath = "//ul[contains(@class,'restaurant-details__classification--list')]//li | //div[contains(@class,'restaurant__classification')]//p[contains(@class,'flex-fill')]"

	// Price extraction XPath
	awardPriceXPath = "//li[contains(@class,'restaurant-details__heading-price') or .//span[contains(@class,'mg-price') or contains(@class,'mg-euro-circle')]] | //div[contains(@class,'data-sheet__block--text')]"

	// Published date extraction XPaths
	awardDateXPath     = "//div[contains(@class,'restaurant-details__heading--label-title')] | //div[contains(@class,'label-text')]"
	awardDateMetaXPath = "//meta[@name='description']"

	// Script tags for dLayer and JSON-LD
	awardScriptXPath = "//script"
	awardJSONLDXPath = "//script[@type='application/ld+json']"
)
