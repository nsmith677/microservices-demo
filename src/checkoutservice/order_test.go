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

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/genproto"
	money "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/money"
)

func usd(units int64, nanos int32) *pb.Money {
	return &pb.Money{CurrencyCode: "USD", Units: units, Nanos: nanos}
}

func eur(units int64, nanos int32) *pb.Money {
	return &pb.Money{CurrencyCode: "EUR", Units: units, Nanos: nanos}
}

func TestSumUSDItemSubtotalUsesCatalogPrices(t *testing.T) {
	items := []*pb.CartItem{
		{ProductId: "watch", Quantity: 1},
	}
	pricesUSD := map[string]*pb.Money{
		"watch": usd(80, 0),
	}

	subtotal := sumUSDItemSubtotal(items, pricesUSD)
	if subtotal.GetCurrencyCode() != "USD" || subtotal.GetUnits() != 80 || subtotal.GetNanos() != 0 {
		t.Fatalf("USD subtotal = %+v, want 80 USD", subtotal)
	}

	localizedLine := eur(70, 760000000)
	if localizedLine.GetUnits() == subtotal.GetUnits() {
		t.Fatalf("localized EUR amount must not be used as USD subtotal threshold input")
	}
}

func TestSumOrderTotalExcludesPaidShippingWhenFree(t *testing.T) {
	orderItems := []*pb.OrderItem{
		{
			Item: &pb.CartItem{ProductId: "watch", Quantity: 1},
			Cost: eur(70, 760000000),
		},
	}
	freeShipping := eur(0, 0)

	total := sumOrderTotal("EUR", freeShipping, orderItems)
	want := money.Must(money.Sum(*freeShipping, money.MultiplySlow(*orderItems[0].Cost, 1)))
	if !money.AreEquals(total, want) {
		t.Fatalf("order total = %+v, want %+v without shipping fee", total, want)
	}
	if total.GetUnits() != 70 || total.GetNanos() != 760000000 {
		t.Fatalf("order total = %+v, expected localized items only", total)
	}
}

func TestSumOrderTotalIncludesPaidShipping(t *testing.T) {
	orderItems := []*pb.OrderItem{
		{
			Item: &pb.CartItem{ProductId: "mug", Quantity: 1},
			Cost: usd(50, 0),
		},
	}
	shipping := usd(8, 990000000)

	total := sumOrderTotal("USD", shipping, orderItems)
	want := money.Must(money.Sum(*shipping, money.MultiplySlow(*orderItems[0].Cost, 1)))
	if !money.AreEquals(total, want) {
		t.Fatalf("order total = %+v, want %+v", total, want)
	}
}
