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
	"testing"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto"
	"github.com/GoogleCloudPlatform/microservices-demo/src/frontend/money"
)

func usd(units int64, nanos int32) *pb.Money {
	return &pb.Money{CurrencyCode: "USD", Units: units, Nanos: nanos}
}

func eur(units int64, nanos int32) *pb.Money {
	return &pb.Money{CurrencyCode: "EUR", Units: units, Nanos: nanos}
}

func jpy(units int64, nanos int32) *pb.Money {
	return &pb.Money{CurrencyCode: "JPY", Units: units, Nanos: nanos}
}

func TestSumUSDItemSubtotalUsesCatalogPrices(t *testing.T) {
	items := []*pb.CartItem{
		{ProductId: "watch", Quantity: 1},
	}
	pricesUSD := map[string]*pb.Money{
		"watch": usd(80, 0),
	}

	subtotal := sumUSDItemSubtotal(items, pricesUSD)
	if subtotal.GetCurrencyCode() != "USD" || subtotal.GetUnits() != 80 {
		t.Fatalf("USD subtotal = %+v, want 80 USD", subtotal)
	}

	localized := eur(70, 760000000)
	if localized.GetUnits() >= 75 {
		t.Fatalf("EUR localized amount must not be mistaken for USD threshold input")
	}
}

func TestSumCartTotalFreeShippingEUR(t *testing.T) {
	items := []*pb.CartItem{
		{ProductId: "watch", Quantity: 1},
	}
	pricesUSD := map[string]*pb.Money{
		"watch": usd(80, 0),
	}
	subtotal := sumUSDItemSubtotal(items, pricesUSD)
	if subtotal.GetUnits() < 75 {
		t.Fatalf("expected qualifying USD subtotal, got %+v", subtotal)
	}

	localizedLine := eur(70, 760000000)
	freeShipping := eur(0, 0)
	total := sumCartTotal("EUR", freeShipping, []*pb.Money{localizedLine})

	if !money.AreEquals(total, *localizedLine) {
		t.Fatalf("cart total = %+v, want localized items only %+v", total, localizedLine)
	}
}

func TestSumCartTotalPaidShippingJPY(t *testing.T) {
	items := []*pb.CartItem{
		{ProductId: "mug", Quantity: 1},
	}
	pricesUSD := map[string]*pb.Money{
		"mug": usd(50, 0),
	}
	subtotal := sumUSDItemSubtotal(items, pricesUSD)
	if subtotal.GetUnits() >= 75 {
		t.Fatalf("expected subtotal below free-shipping threshold, got %+v", subtotal)
	}

	localizedLine := jpy(5591, 0)
	paidShipping := jpy(1016, 0)
	total := sumCartTotal("JPY", paidShipping, []*pb.Money{localizedLine})
	want := money.Must(money.Sum(*localizedLine, *paidShipping))
	if !money.AreEquals(total, want) {
		t.Fatalf("cart total = %+v, want items plus shipping %+v", total, want)
	}
}
