package bean

import "time"

type Position struct {
	*Contract
	qty   float64
	price float64
}

func (p Position) Qty() float64 {
	return p.qty
}

func (p Position) Price() float64 {
	return p.price
}

func PositionsFromNames(names []string, quantities []float64, prices []float64) (posns []Position, err error) {
	var c *Contract
	posns = make([]Position, 0)
	for i := range names {
		c, err = ContractFromName(names[i])
		if err != nil {
			return
		}
		var p Position
		if prices == nil || quantities == nil {
			p = NewPosition(c, 0.0, 0.0)
		} else {
			p = NewPosition(c, quantities[i], prices[i])
		}
		posns = append(posns, p)
	}
	return
}

func NewPosition(c *Contract, qty, price float64) Position {
	return Position{Contract: c, qty: qty, price: price}
}

// Calculate the price of a contract given market parameters. Price is in RHS coin value spot
// Discounting assumes zero interest rate on LHS coin (normally BTC) which is deribit standard. Note USD rates float and are generally negative.
func (p Position) PV(asof time.Time, spotPrice, futPrice, vol float64) float64 {
	if p.IsOption() {
		/*		return p.Con.OptPrice(asof, spotPrice, futPrice, vol) * p.Qty*/
		return p.OptPrice(asof, spotPrice, futPrice, vol)*p.qty - p.price*spotPrice*p.qty
	} else {
		return (1.0/p.price - 1.0/futPrice) * spotPrice * p.qty * 10.0
	}
}

// in rhs coin spot value
func (p Position) Vega(asof time.Time, spotPrice, futPrice, vol float64) float64 {
	return p.PV(asof, spotPrice, futPrice, vol+0.005) - p.PV(asof, spotPrice, futPrice, vol-0.005)
}

//in lhs coin spot value
func (p Position) Delta(asof time.Time, spotPrice, futPrice, vol float64) float64 {
	deltaFiat := (p.PV(asof, spotPrice*1.005, futPrice*1.005, vol) - p.PV(asof, spotPrice*0.995, futPrice*0.995, vol)) * 100.0

	return deltaFiat / spotPrice
}

func (p Position) BucketDelta(asof time.Time, spotPrice, futPrice, vol float64) map[string]float64 {
	totdelta := (p.PV(asof, spotPrice*1.005, futPrice*1.005, vol) - p.PV(asof, spotPrice*0.995, futPrice*0.995, vol)) * 100.0
	spotDelta := (p.PV(asof, spotPrice*1.005, futPrice, vol) - p.PV(asof, spotPrice*0.995, futPrice, vol)) * 100.0

	delta := make(map[string]float64)
	delta["CASH"] = spotDelta / spotPrice
	delta[p.ExpiryStr()] = (totdelta - spotDelta) / spotPrice

	return delta
}

//in lhs coin spot value
func (p Position) Gamma(asof time.Time, spotPrice, futPrice, vol float64) float64 {
	gammaFiat := p.Delta(asof, spotPrice*1.005, futPrice*1.005, vol) - p.Delta(asof, spotPrice*0.995, futPrice*0.995, vol)

	return gammaFiat
}

//in rhs coin spot value
func (p Position) Theta(asof time.Time, spotPrice, futPrice, vol float64) float64 {
	return p.PV(asof.Add(24*time.Hour), spotPrice, futPrice, vol) - p.PV(asof, spotPrice, futPrice, vol)
}
