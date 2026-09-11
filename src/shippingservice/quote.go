// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"math"
)

const (
	baseShippingRateUSD         = 8.99
	freeShippingThresholdUnits  = 75
	freeShippingThresholdNanos  = 0
	freeShippingThresholdCurrency = "USD"
)

// Quote represents a currency value.
type Quote struct {
	Dollars uint32
	Cents   uint32
}

// String representation of the Quote.
func (q Quote) String() string {
	return fmt.Sprintf("$%d.%d", q.Dollars, q.Cents)
}

// CreateQuoteFromCount takes a number of items and returns a shipping quote.
func CreateQuoteFromCount(count int) Quote {
	return CreateQuote(count, 0, 0, "")
}

// CreateQuote returns a shipping quote based on item count and optional USD subtotal.
func CreateQuote(count int, usdSubtotalUnits int64, usdSubtotalNanos int32, usdSubtotalCurrency string) Quote {
	if count == 0 {
		return CreateQuoteFromFloat(0)
	}
	if qualifiesForFreeShipping(usdSubtotalUnits, usdSubtotalNanos, usdSubtotalCurrency) {
		return CreateQuoteFromFloat(0)
	}
	return CreateQuoteFromFloat(baseShippingRateUSD)
}

func qualifiesForFreeShipping(units int64, nanos int32, currency string) bool {
	if currency != freeShippingThresholdCurrency {
		return false
	}
	return moneyGTE(units, nanos, freeShippingThresholdUnits, freeShippingThresholdNanos)
}

func moneyGTE(units int64, nanos int32, thresholdUnits int64, thresholdNanos int32) bool {
	if units > thresholdUnits {
		return true
	}
	if units < thresholdUnits {
		return false
	}
	return nanos >= thresholdNanos
}

// CreateQuoteFromFloat takes a price represented as a float and creates a Price struct.
func CreateQuoteFromFloat(value float64) Quote {
	units, fraction := math.Modf(value)
	return Quote{
		uint32(units),
		uint32(math.Trunc(fraction * 100)),
	}
}
