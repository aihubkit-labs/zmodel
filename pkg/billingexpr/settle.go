package billingexpr

import "github.com/QuantumNous/new-api/common"

// quotaConversion converts raw expression output to quota based on the
// expression version. This is the central dispatch point for future versions
// that may use a different conversion formula.
func quotaConversion(exprOutput float64, snap *BillingSnapshot) float64 {
	switch snap.ExprVersion {
	default: // v1: coefficients are $/1M tokens prices
		return exprOutput / 1_000_000 * snap.QuotaPerUnit
	}
}

// ComputeTieredQuota runs the Expr from a frozen BillingSnapshot against
// actual token counts and returns the settlement result.
func ComputeTieredQuota(snap *BillingSnapshot, params TokenParams) (TieredResult, error) {
	return ComputeTieredQuotaWithDimensionsAndRequest(snap, params, snap.EstimatedDimensions, RequestInput{})
}

func ComputeTieredQuotaWithRequest(snap *BillingSnapshot, params TokenParams, request RequestInput) (TieredResult, error) {
	return ComputeTieredQuotaWithDimensionsAndRequest(snap, params, snap.EstimatedDimensions, request)
}

func ComputeTieredQuotaWithDimensionsAndRequest(snap *BillingSnapshot, params TokenParams, dimensions BillingDimensions, request RequestInput) (TieredResult, error) {
	if err := ValidateBillingDimensions(dimensions, UsedVars(snap.ExprString)); err != nil {
		return TieredResult{}, err
	}
	cost, trace, err := RunExprByHashWithDimensionsAndRequest(snap.ExprString, snap.ExprHash, params, dimensions, request)
	if err != nil {
		return TieredResult{}, err
	}

	quotaBeforeGroup := quotaConversion(cost, snap)
	afterGroup, clamp := common.QuotaRoundChecked(quotaBeforeGroup * snap.GroupRatio)
	crossed := trace.MatchedTier != snap.EstimatedTier

	return TieredResult{
		ActualQuotaBeforeGroup: quotaBeforeGroup,
		ActualQuotaAfterGroup:  afterGroup,
		MatchedTier:            trace.MatchedTier,
		CrossedTier:            crossed,
		ActualDimensions:       dimensions,
		Clamp:                  clamp,
	}, nil
}
