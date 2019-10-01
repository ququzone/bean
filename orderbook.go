package bean

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// OrderBook extends the OrderBookCore functions (insert, cancel, edit, retrieve)
// with additional functions (spread, equality, depth) independent of the core implementation
type OrderBook struct {
	OrderBookCore
}

// OrderBookT extends the orderbook with a timestamp of the last update
type OrderBookT struct {
	OrderBook
	Time     time.Time
	ChangeId int64
}

// OrderBookTS is a timeseries of orderbooks each with their own timestamp
type OrderBookTS []OrderBookT

// Order represents a single resting order on an exchange
type Order struct {
	Price  float64
	Amount float64
}

// OrderBookCore defines the core functions needed in the OrderBook object
// These are implemented in the OrderBook1 array implementation (OrderBook2 map implementation pending)
type OrderBookCore interface {
	Bids() []Order        // Bids returns a list of live orders
	Asks() []Order        // Asks returns a list of live orders
	InsertBid(Order) bool // InsertBid inserts a new order into the orderbook
	InsertAsk(Order) bool // InsertAsk insterts a new order into the orderbook
	CancelBid(Order) bool // CancelBid removes an order from the orderbook
	CancelAsk(Order) bool // CancelAsk removes an order from the orderbook
	EditBid(Order) bool   // Edit replaces an order with another one at the same level
	EditAsk(Order) bool   // Edit replaces an order with another one at the same level
	BestBid() Order       // Bestbid returns the top of the orderbook
	BestAsk() Order       // Bestask returns the top of the orderbook
}

// OrderBook1 is an implementation of the OrderBookCore interface. Bids and asks are stored as lists of orders
type OrderBook1 struct {
	bids []Order
	asks []Order
	m    sync.Mutex
}

// Bids retrieves a list of bid orders from the orderbook
func (ob *OrderBook1) Bids() []Order {
	return ob.bids
}

// Asks retrieves a list of asks from the orderbook
func (ob *OrderBook1) Asks() []Order {
	return ob.asks
}

// EmptyOrderBook returns an empty orderbook
func EmptyOrderBook() OrderBook {
	return OrderBook{new(OrderBook1)}
}

// NewOrderBook returns a new order book populated by bids and offers
func NewOrderBook(bids, asks []Order) OrderBook {
	ob := OrderBook1{bids: bids, asks: asks}.Sort()
	return OrderBook{&ob}
}

// InsertBid adds a new order into the orderbook. Returns true if the top of book price has changed
func (ob *OrderBook1) InsertBid(order Order) (tob bool) {
	ob.m.Lock()
	defer ob.m.Unlock()
	ob.bids = append(ob.bids, order)
	ob.Sort()
	return order.Price == ob.bids[0].Price
}

// InsertAsk adds a new order into the orderbook. Returns true if the top of book price has changed
func (ob *OrderBook1) InsertAsk(order Order) (tob bool) {
	ob.m.Lock()
	defer ob.m.Unlock()
	ob.asks = append(ob.asks, order)
	ob.Sort()
	return order.Price == ob.asks[0].Price
}

// CancelBid deletes an order from the orderbook. Returns true if the top of book price has changed
func (ob *OrderBook1) CancelBid(order Order) (tob bool) {
	ob.m.Lock()
	defer ob.m.Unlock()
	for i := range ob.bids {
		if ob.bids[i].Price == order.Price {
			ob.bids = append(ob.bids[:i], ob.bids[i+1:]...)
			if i == 0 {
				tob = true
			}
			break
		}
	}
	return
}

// CancelAsk deletes an order from the orderbook. Returns true if the top of book price has changed
func (ob *OrderBook1) CancelAsk(order Order) (tob bool) {
	ob.m.Lock()
	defer ob.m.Unlock()
	for i := range ob.asks {
		if ob.asks[i].Price == order.Price {
			ob.asks = append(ob.asks[:i], ob.asks[i+1:]...)
			if i == 0 {
				tob = true
			}
			break
		}
	}
	return
}

// EditBid replaces an order at a particular level with another. Returns true if the top of book has changed
func (ob *OrderBook1) EditBid(order Order) (tob bool) {
	ob.m.Lock()
	defer ob.m.Unlock()
	for i := range ob.bids {
		if ob.bids[i].Price == order.Price {
			ob.bids[i].Amount = order.Amount
			break
		}
	}
	return
}

// EditAsk replaces an order at a particular level with another. Returns true if the top of book has changed
func (ob *OrderBook1) EditAsk(order Order) (tob bool) {
	ob.m.Lock()
	defer ob.m.Unlock()
	for i := range ob.asks {
		if ob.asks[i].Price == order.Price {
			ob.asks[i].Amount = order.Amount
			break
		}
	}
	return
}

func (ob *OrderBook1) BestBid() Order {
	if ob != nil && len(ob.bids) > 0 {
		return ob.bids[0]
	} else {
		return Order{Price: math.NaN(), Amount: 0.0}
	}
}

func (ob *OrderBook1) BestAsk() Order {
	if ob != nil && len(ob.asks) > 0 {
		return ob.asks[0]
	} else {
		return Order{Price: math.NaN(), Amount: 0.0}
	}
}
func (ob OrderBook1) Sort() OrderBook1 {
	// asks in ascending order
	sort.Slice(ob.asks, func(i, j int) bool { return ob.asks[i].Price < ob.asks[j].Price })
	// bids in descending order
	sort.Slice(ob.bids, func(i, j int) bool { return ob.bids[i].Price > ob.bids[j].Price })
	return ob
}

func (ob *OrderBook) BidAskMid() (bid, ask, mid float64) {
	if ob == nil {
		bid, ask, mid = math.NaN(), math.NaN(), math.NaN()
		return
	}
	bid = ob.BestBid().Price
	ask = ob.BestAsk().Price
	if math.IsNaN(ask) {
		mid = bid
	} else if math.IsNaN(bid) {
		mid = ask
	} else {
		mid = (bid + ask) / 2.0
	}
	return
}

func (ob *OrderBook) Spread() float64 {
	return ob.BestAsk().Price - ob.BestBid().Price
}

func (ob *OrderBook) Empty() bool {
	return ob.BestBid().Amount == 0.0 && ob.BestAsk().Amount == 0.0
}

func (ob *OrderBook) Valid() bool {
	return !math.IsNaN(ob.BestBid().Price) && !math.IsNaN(ob.BestAsk().Price)
}

func (ob *OrderBook) Copy() OrderBook {
	bids := ob.Bids()
	asks := ob.Asks()
	ob2 := OrderBook1{
		bids: make([]Order, len(bids)),
		asks: make([]Order, len(asks)),
	}
	for i := range bids {
		ob2.bids[i] = bids[i]
	}
	for i := range asks {
		ob2.asks[i] = asks[i]
	}
	return OrderBook{&ob2}
}
func (ob *OrderBook) Mid() float64 {
	return (ob.BestBid().Price + ob.BestAsk().Price) / 2.0
}

func (obt *OrderBookT) Copy() *OrderBookT {
	return &OrderBookT{
		OrderBook: obt.OrderBook.Copy(),
		Time:      obt.Time,
	}
}

// Compare two orderbooks. Equal if the best bid and best offer hasn't changed
func (ob1 *OrderBook) Equal(ob2 *OrderBook) bool {
	return (!ob1.Valid() && !ob2.Valid()) ||
		(ob1.BestBid() == ob2.BestBid() && ob1.BestAsk() == ob2.BestAsk())
}

// filter out orders with amount less than the Coin minimum trading amount
// assuming ob is sorted
func (ob *OrderBook) Denoise(pair Pair) *OrderBook {
	var bids []Order
	var asks []Order
	minimumAmount := pair.MinimumTradingAmount()
	carryAmount := 0.0
	for _, b := range ob.Bids() {
		if b.Amount+carryAmount < minimumAmount {
			carryAmount += b.Amount
		} else {
			b.Amount += carryAmount
			bids = append(bids, b)
		}
	}

	for _, b := range ob.Asks() {
		if b.Amount+carryAmount < minimumAmount {
			carryAmount += b.Amount
		} else {
			b.Amount += carryAmount
			asks = append(asks, b)
		}
	}
	ob2 := NewOrderBook(bids, asks)
	return &ob2
}

// OrderBook display
func (ob OrderBook) ShowBrief() {
	if ob.Valid() {
		fmt.Println("depth:", len(ob.Asks()), "bestBid:", ob.BestBid().Price, "bestAsk:", ob.BestAsk().Price)
	} else {
		fmt.Println("empty orderbook")
	}
}

// ShowBrief prints a summary of the orderbook.
func (ob OrderBookT) ShowBrief() {
	ob.OrderBook.ShowBrief()
	fmt.Println("Timestamp: " + ob.Time.Local().Format(time.ANSIC))
}

// OrderBookTS display
func (obts OrderBookTS) ShowBrief() {
	for _, ob := range obts {
		ob.ShowBrief()
	}
}

// Sort sorts a timesliced orderbook
func (obts OrderBookTS) Sort() OrderBookTS {
	sort.Slice(obts, func(i, j int) bool { return obts[i].Time.Before(obts[j].Time) })
	return obts
}

// return the orderbook of time t (the closest in sample), assuming the obts is sorted
func (obts OrderBookTS) GetOrderBook(t time.Time) *OrderBookT {
	ob := &obts[0]
	for i := range obts {
		if t.After(obts[i].Time) {
			ob = &obts[i]
		} else {
			break
		}
	}
	return ob
}

// PriceIn returns the worst bid and worst ask that need to be hit in the orderbook in order to execute a requested size
// Also returns the total size available at that price (may be more than requested size)
// If orderstack does not have sufficient liquidity, then it returns the size available
func (ob OrderBook) PriceIn(size float64) (bid, ask, bidSize, askSize float64) {
	bid, bidSize = ob.BidIn(size)
	ask, askSize = ob.AskIn(size)
	return
}

// BidIn returns the worst bid that needs to be hit in order to fill a target size. If insufficient liquidity in the stack
// then available is the maximum in the stack
func (ob OrderBook) BidIn(size float64) (price, available float64) {
	price, available = priceInAmount(size, ob.Bids())
	return
}

// AskIn returns the worst bid that needs to be hit in order to fill a target size. If insufficient liquidity in the stack
// then available is the maximum in the stack
func (ob OrderBook) AskIn(size float64) (price, available float64) {
	price, available = priceInAmount(size, ob.Asks())
	return
}

func priceInAmount(requiredAmount float64, stack []Order) (price, available float64) {
	available = 0.0

	if len(stack) == 0 {
		price = math.NaN()
		return
	}

	for _, ord := range stack {
		available += ord.Amount
		if available > requiredAmount {
			price = ord.Price
			return
		}
	}
	price = stack[len(stack)-1].Price
	return
}

// SBRatio ... sell / buy ratio, alpha in (0, 1]
func (ob OrderBook) SBRatio(alpha float64) float64 {
	var sell float64
	var buy float64
	if ob.Valid() {
		// FIXME: generalize spread, work for IOTX at the moment
		sprd := (ob.BestAsk().Price - ob.BestBid().Price) * 1e8

		for i, v := range ob.Asks() {
			if i == 10 {
				break
			} else {
				sell += math.Pow(alpha, float64(i)) * v.Price * v.Amount
			}
		}
		for i, v := range ob.Bids() {
			if i == 10 {
				break
			} else {
				buy += math.Pow(alpha, sprd-1+float64(i)) * v.Price * v.Amount
			}
		}
	}
	return sell / buy
}

// Match ... Takes a placed order and matches against the existing orderbook.
// If it can be filled then the filled amount and rate are returned
// Orders (aggressor) are filled at the orderbook (market maker) rate
func (ob OrderBook) Match(placedOrder Order) Order {
	fillCounterAmount := 0.0
	fillAmount := 0.0
	if placedOrder.Amount > 0.0 {
		for _, o := range ob.Asks() {
			if o.Price <= placedOrder.Price {
				fillCounterAmount += math.Min(placedOrder.Amount-fillAmount, o.Amount) * o.Price
				fillAmount += math.Min(placedOrder.Amount-fillAmount, o.Amount)
			}
		}
		if fillAmount > 0.0 {
			return Order{Price: fillCounterAmount / fillAmount, Amount: fillAmount}
		} else {
			return Order{Price: 0.0, Amount: 0.0}
		}
	} else {
		for _, o := range ob.Bids() {
			if o.Price >= placedOrder.Price {
				fillCounterAmount += math.Min(-placedOrder.Amount-fillAmount, o.Amount) * o.Price
				fillAmount += math.Min(-placedOrder.Amount-fillAmount, o.Amount)
			}
		}
		if fillAmount > 0.0 {
			return Order{Price: fillCounterAmount / fillAmount, Amount: -fillAmount}
		} else {
			return Order{Price: 0.0, Amount: 0.0}
		}
	}
}

type OrderState string

const (
	ALIVE     OrderState = "ALIVE"
	FILLED    OrderState = "FILLED"
	CANCELLED OrderState = "CANCELLED"
)

type Side string

const (
	BUY  Side = "BUY"
	SELL Side = "SELL"
)

func AmountToSide(amt float64) Side {
	if amt < 0.0 {
		return SELL
	} else {
		return BUY
	}
}
