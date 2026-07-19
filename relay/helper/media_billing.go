package helper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
)

const maxBillingImageDimension = 5120

func BuildImageBillingDimensions(request *dto.ImageRequest) (billingexpr.BillingDimensions, error) {
	if request == nil {
		return billingexpr.BillingDimensions{}, fmt.Errorf("image request is required")
	}

	units := uint(1)
	if request.N != nil {
		units = *request.N
	}
	if units == 0 || units > dto.MaxImageN {
		return billingexpr.BillingDimensions{}, fmt.Errorf("n must be an integer between 1 and %d", dto.MaxImageN)
	}

	dimensions := billingexpr.BillingDimensions{
		Units:   float64(units),
		Quality: strings.ToLower(strings.TrimSpace(request.Quality)),
	}

	size := strings.ToLower(strings.Join(strings.Fields(request.Size), ""))
	if size == "" || size == "auto" {
		dimensions.ImageSize = size
		dimensions.ImageSizeTier = "2K"
		return dimensions, nil
	}
	if size == "1k" || size == "2k" || size == "4k" {
		dimensions.ImageSize = strings.ToUpper(size)
		dimensions.ImageSizeTier = strings.ToUpper(size)
		return dimensions, nil
	}

	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		dimensions.ImageSize = size
		dimensions.ImageSizeTier = "2K"
		return dimensions, nil
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		dimensions.ImageSize = size
		dimensions.ImageSizeTier = "2K"
		return dimensions, nil
	}
	if width <= 0 || width > maxBillingImageDimension {
		return billingexpr.BillingDimensions{}, fmt.Errorf("image width must be between 1 and %d", maxBillingImageDimension)
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		dimensions.ImageSize = size
		dimensions.ImageSizeTier = "2K"
		return dimensions, nil
	}
	if height <= 0 || height > maxBillingImageDimension {
		return billingexpr.BillingDimensions{}, fmt.Errorf("image height must be between 1 and %d", maxBillingImageDimension)
	}

	dimensions.Width = float64(width)
	dimensions.Height = float64(height)
	dimensions.ImageSize = fmt.Sprintf("%dx%d", width, height)
	maxSide := max(width, height)
	switch {
	case maxSide <= 1024:
		dimensions.ImageSizeTier = "1K"
	case maxSide <= 2048:
		dimensions.ImageSizeTier = "2K"
	case maxSide <= 5120:
		dimensions.ImageSizeTier = "4K"
	}
	return dimensions, nil
}
