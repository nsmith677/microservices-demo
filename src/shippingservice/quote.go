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

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/shippingservice/genproto"
)

const freeShippingThresholdUSDUnits int64 = 75

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
	if count == 0 {
		return CreateQuoteFromFloat(0)
	}
	return CreateQuoteFromFloat(8.99)
}

// qualifiesForFreeShipping reports whether a catalog subtotal earns free shipping.
// The threshold is $75.00 USD inclusive. Non-USD or missing amounts never qualify,
// so a shopper's display currency cannot accidentally meet the threshold.
func qualifiesForFreeShipping(subtotal *pb.Money) bool {
	if subtotal == nil {
		return false
	}
	if subtotal.GetCurrencyCode() != "USD" {
		return false
	}
	units := subtotal.GetUnits()
	nanos := subtotal.GetNanos()
	if units > freeShippingThresholdUSDUnits {
		return true
	}
	if units == freeShippingThresholdUSDUnits && nanos >= 0 {
		return true
	}
	return false
}

func quoteUSD(q Quote) *pb.Money {
	return &pb.Money{
		CurrencyCode: "USD",
		Units:        int64(q.Dollars),
		Nanos:        int32(q.Cents * 10000000),
	}
}

// CreateQuoteFromFloat takes a price represented as a float and creates a Price struct.
func CreateQuoteFromFloat(value float64) Quote {
	units, fraction := math.Modf(value)
	return Quote{
		uint32(units),
		uint32(math.Trunc(fraction * 100)),
	}
}