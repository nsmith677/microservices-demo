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
	"regexp"
	"testing"

	"golang.org/x/net/context"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/shippingservice/genproto"
)

// TestGetQuote is a basic check on the GetQuote RPC service.
func TestGetQuote(t *testing.T) {
	s := server{}

	// A basic test case to test logic and protobuf interactions.
	req := &pb.GetQuoteRequest{
		Address: &pb.Address{
			StreetAddress: "Muffin Man",
			City:          "London",
			State:         "",
			Country:       "England",
		},
		Items: []*pb.CartItem{
			{
				ProductId: "23",
				Quantity:  1,
			},
			{
				ProductId: "46",
				Quantity:  3,
			},
		},
	}

	res, err := s.GetQuote(context.Background(), req)
	if err != nil {
		t.Errorf("TestGetQuote (%v) failed", err)
	}
	if res.CostUsd.GetUnits() != 8 || res.CostUsd.GetNanos() != 990000000 {
		t.Errorf("TestGetQuote: Quote value '%d.%d' does not match expected '8.990000000'", res.CostUsd.GetUnits(), res.CostUsd.GetNanos())
	}
}

func usd(units int64, nanos int32) *pb.Money {
	return &pb.Money{CurrencyCode: "USD", Units: units, Nanos: nanos}
}

func quoteItems() []*pb.CartItem {
	return []*pb.CartItem{
		{ProductId: "23", Quantity: 1},
		{ProductId: "46", Quantity: 3},
	}
}

func TestGetQuoteFreeShippingThreshold(t *testing.T) {
	s := server{}
	tests := []struct {
		name      string
		subtotal  *pb.Money
		wantUnits int64
		wantNanos int32
	}{
		{"usd exactly 75", usd(75, 0), 0, 0},
		{"usd 75 plus one nano", usd(75, 1), 0, 0},
		{"usd above 75", usd(80, 0), 0, 0},
		{"usd 74.99", usd(74, 990000000), 8, 990000000},
		{"eur 80 must not qualify", &pb.Money{CurrencyCode: "EUR", Units: 80}, 8, 990000000},
		{"jpy 10000 must not qualify", &pb.Money{CurrencyCode: "JPY", Units: 10000}, 8, 990000000},
		{"nil subtotal", nil, 8, 990000000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := s.GetQuote(context.Background(), &pb.GetQuoteRequest{
				Address:     &pb.Address{StreetAddress: "1 Main", City: "London", Country: "England"},
				Items:       quoteItems(),
				SubtotalUsd: tc.subtotal,
			})
			if err != nil {
				t.Fatalf("GetQuote: %v", err)
			}
			if res.CostUsd.GetUnits() != tc.wantUnits || res.CostUsd.GetNanos() != tc.wantNanos {
				t.Errorf("quote = %d.%d, want %d.%d", res.CostUsd.GetUnits(), res.CostUsd.GetNanos(), tc.wantUnits, tc.wantNanos)
			}
			if res.CostUsd.GetCurrencyCode() != "USD" {
				t.Errorf("currency = %q, want USD", res.CostUsd.GetCurrencyCode())
			}
		})
	}
}

func TestQualifiesForFreeShippingRejectsWrongCurrency(t *testing.T) {
	if qualifiesForFreeShipping(&pb.Money{CurrencyCode: "EUR", Units: 80}) {
		t.Fatal("EUR 80 must not qualify; threshold is USD-only")
	}
	if !qualifiesForFreeShipping(usd(75, 0)) {
		t.Fatal("USD 75 must qualify")
	}
}

// TestGetQuoteEmptyCart verifies that an empty cart returns a zero quote.
func TestGetQuoteEmptyCart(t *testing.T) {
	s := server{}

	req := &pb.GetQuoteRequest{
		Address: &pb.Address{
			StreetAddress: "221B Baker Street",
			City:          "London",
			State:         "",
			Country:       "England",
		},
		Items: []*pb.CartItem{},
	}

	res, err := s.GetQuote(context.Background(), req)
	if err != nil {
		t.Errorf("TestGetQuoteEmptyCart (%v) failed", err)
	}
	if res.CostUsd.GetUnits() != 0 || res.CostUsd.GetNanos() != 0 {
		t.Errorf("TestGetQuoteEmptyCart: expected zero quote for empty cart, got '%d.%d'", res.CostUsd.GetUnits(), res.CostUsd.GetNanos())
	}
}

// TestShipOrder is a basic check on the ShipOrder RPC service.
func TestShipOrder(t *testing.T) {
	s := server{}

	// A basic test case to test logic and protobuf interactions.
	req := &pb.ShipOrderRequest{
		Address: &pb.Address{
			StreetAddress: "Muffin Man",
			City:          "London",
			State:         "",
			Country:       "England",
		},
		Items: []*pb.CartItem{
			{
				ProductId: "23",
				Quantity:  1,
			},
			{
				ProductId: "46",
				Quantity:  3,
			},
		},
	}

	res, err := s.ShipOrder(context.Background(), req)
	if err != nil {
		t.Errorf("TestShipOrder (%v) failed", err)
	}
	if len(res.TrackingId) != 18 {
		t.Errorf("TestShipOrder: Tracking ID is malformed - has %d characters, %d expected", len(res.TrackingId), 18)
	}
}

// TestTrackingIdFormat verifies the tracking ID matches the expected pattern.
func TestTrackingIdFormat(t *testing.T) {
	pattern := regexp.MustCompile(`^[A-Z]{2}-\d+-\d+$`)

	for i := 0; i < 20; i++ {
		id := CreateTrackingId("test-salt-value")
		if !pattern.MatchString(id) {
			t.Errorf("CreateTrackingId: '%s' does not match expected pattern '[A-Z]{2}-\\d+-\\d+'", id)
		}
	}
}

// TestTrackingIdUniqueness checks that generated IDs are not all identical.
func TestTrackingIdUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := CreateTrackingId("same-salt")
		seen[id] = true
	}
	if len(seen) < 2 {
		t.Errorf("CreateTrackingId: expected unique IDs but got %d distinct values out of 50", len(seen))
	}
}

// TestCreateQuoteFromFloat verifies quote creation from float values.
func TestCreateQuoteFromFloat(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		dollars uint32
		cents   uint32
	}{
		{"zero", 0.0, 0, 0},
		{"whole dollars", 10.0, 10, 0},
		{"with cents", 8.99, 8, 99},
		{"small value", 0.50, 0, 50},
		{"large value", 100.01, 100, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := CreateQuoteFromFloat(tc.value)
			if q.Dollars != tc.dollars || q.Cents != tc.cents {
				t.Errorf("CreateQuoteFromFloat(%v) = $%d.%d, want $%d.%d",
					tc.value, q.Dollars, q.Cents, tc.dollars, tc.cents)
			}
		})
	}
}

// TestCreateQuoteFromCount verifies count-based quote generation.
func TestCreateQuoteFromCount(t *testing.T) {
	zeroQuote := CreateQuoteFromCount(0)
	if zeroQuote.Dollars != 0 || zeroQuote.Cents != 0 {
		t.Errorf("CreateQuoteFromCount(0) = %s, want $0.0", zeroQuote)
	}

	nonZeroQuote := CreateQuoteFromCount(5)
	if nonZeroQuote.Dollars == 0 && nonZeroQuote.Cents == 0 {
		t.Error("CreateQuoteFromCount(5) returned zero, expected a non-zero quote")
	}
}

// TestGetRandomLetterCode verifies the output is a valid uppercase letter.
func TestGetRandomLetterCode(t *testing.T) {
	for i := 0; i < 100; i++ {
		code := getRandomLetterCode()
		if code < 65 || code > 90 {
			t.Errorf("getRandomLetterCode: got %d (%c), expected range 65-90 (A-Z)", code, code)
		}
	}
}

// TestGetRandomNumber verifies the output has the correct number of digits.
func TestGetRandomNumber(t *testing.T) {
	for _, digits := range []int{1, 3, 5, 7, 10} {
		result := getRandomNumber(digits)
		if len(result) != digits {
			t.Errorf("getRandomNumber(%d) = '%s' (len %d), expected length %d",
				digits, result, len(result), digits)
		}
	}
}

// TestQuoteString verifies the string representation of a Quote.
func TestQuoteString(t *testing.T) {
	q := Quote{Dollars: 8, Cents: 99}
	expected := "$8.99"
	if q.String() != expected {
		t.Errorf("Quote.String() = '%s', want '%s'", q.String(), expected)
	}
}
