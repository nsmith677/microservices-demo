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
	pb "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/genproto"
	money "github.com/GoogleCloudPlatform/microservices-demo/src/checkoutservice/money"
)

func sumUSDItemSubtotal(items []*pb.CartItem, pricesUSD map[string]*pb.Money) *pb.Money {
	subtotal := pb.Money{CurrencyCode: usdCurrency, Units: 0, Nanos: 0}
	for _, item := range items {
		priceUSD := pricesUSD[item.GetProductId()]
		lineUSD := money.MultiplySlow(*priceUSD, uint32(item.GetQuantity()))
		subtotal = money.Must(money.Sum(subtotal, lineUSD))
	}
	return &subtotal
}

func sumOrderTotal(currency string, shipping *pb.Money, orderItems []*pb.OrderItem) pb.Money {
	total := pb.Money{CurrencyCode: currency, Units: 0, Nanos: 0}
	total = money.Must(money.Sum(total, *shipping))
	for _, it := range orderItems {
		linePrice := money.MultiplySlow(*it.Cost, uint32(it.GetItem().GetQuantity()))
		total = money.Must(money.Sum(total, linePrice))
	}
	return total
}
