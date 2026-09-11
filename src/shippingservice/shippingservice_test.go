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

var testCartItems = []*pb.CartItem{
	{ProductId: "23", Quantity: 1},
	{ProductId: "46", Quantity: 3},
}

func assertQuoteUSD(t *testing.T, res *pb.GetQuoteResponse, wantUnits int64, wantNanos int32) {
	t.Helper()
	if res.CostUsd.GetUnits() != wantUnits || res.CostUsd.GetNanos() != wantNanos {
		t.Errorf("quote = %d.%09d USD, want %d.%09d USD",
			res.CostUsd.GetUnits(), res.CostUsd.GetNanos(), wantUnits, wantNanos)
	}
}

func usdMoney(units int64, nanos int32) *pb.Money {
	return &pb.Money{CurrencyCode: "USD", Units: units, Nanos: nanos}
}

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
		Items:           testCartItems,
		UsdItemSubtotal: usdMoney(50, 0),
	}

	res, err := s.GetQuote(context.Background(), req)
	if err != nil {
		t.Errorf("TestGetQuote (%v) failed", err)
	}
	if res.CostUsd.GetUnits() != 8 || res.CostUsd.GetNanos() != 990000000 {
		t.Errorf("TestGetQuote: Quote value '%d.%d' does not match expected '8.990000000'", res.CostUsd.GetUnits(), res.CostUsd.GetNanos())
	}
}

func TestGetQuoteFreeShippingAtThreshold(t *testing.T) {
	s := server{}
	req := &pb.GetQuoteRequest{
		Items:           testCartItems,
		UsdItemSubtotal: usdMoney(75, 0),
	}
	res, err := s.GetQuote(context.Background(), req)
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}
	assertQuoteUSD(t, res, 0, 0)
}

func TestGetQuotePaidShippingBelowThreshold(t *testing.T) {
	s := server{}
	req := &pb.GetQuoteRequest{
		Items:           testCartItems,
		UsdItemSubtotal: usdMoney(74, 990000000),
	}
	res, err := s.GetQuote(context.Background(), req)
	if err != nil {
		t.Fatalf("GetQuote failed: %v", err)
	}
	assertQuoteUSD(t, res, 8, 990000000)
}

func TestGetQuoteIgnoresNonUsdSubtotal(t *testing.T) {
	s := server{}
	tests := []struct {
		name     string
		subtotal *pb.Money
	}{
		{
			name:     "EUR subtotal that looks over 75",
			subtotal: &pb.Money{CurrencyCode: "EUR", Units: 80, Nanos: 0},
		},
		{
			name:     "JPY subtotal that looks over 75",
			subtotal: &pb.Money{CurrencyCode: "JPY", Units: 5000, Nanos: 0},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &pb.GetQuoteRequest{
				Items:           testCartItems,
				UsdItemSubtotal: tc.subtotal,
			}
			res, err := s.GetQuote(context.Background(), req)
			if err != nil {
				t.Fatalf("GetQuote failed: %v", err)
			}
			assertQuoteUSD(t, res, 8, 990000000)
		})
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

func TestCreateQuoteFreeShippingThreshold(t *testing.T) {
	tests := []struct {
		name     string
		units    int64
		nanos    int32
		currency string
		wantZero bool
	}{
		{"USD at threshold", 75, 0, "USD", true},
		{"USD above threshold", 80, 0, "USD", true},
		{"USD below threshold", 74, 990000000, "USD", false},
		{"EUR ignored", 80, 0, "EUR", false},
		{"JPY ignored", 5000, 0, "JPY", false},
		{"missing currency", 80, 0, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := CreateQuote(1, tc.units, tc.nanos, tc.currency)
			gotZero := q.Dollars == 0 && q.Cents == 0
			if gotZero != tc.wantZero {
				t.Errorf("CreateQuote() = %s, wantZero=%v", q, tc.wantZero)
			}
		})
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
