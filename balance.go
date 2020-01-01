package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/adshao/go-binance"
)

func byLockedBalance(c1, c2 *binance.Balance) bool {
	return strings.Compare(c1.Locked, c2.Locked) > 0
}

func byFreeBalance(c1, c2 *binance.Balance) bool {
	f1, _ := strconv.ParseFloat(c1.Free, 64)
	f2, _ := strconv.ParseFloat(c2.Free, 64)
	return f1 > f2
	// return strings.Compare(c1.Free, c2.Free) > 0
}

func byTotalBalance(c1, c2 *binance.Balance) bool {
	f1, err := strconv.ParseFloat(c1.Free, 64)
	if err != nil {
		fmt.Println(err)
		return false
	}

	l1, err := strconv.ParseFloat(c1.Locked, 64)
	if err != nil {
		fmt.Println(err)
		return false
	}

	f2, err := strconv.ParseFloat(c2.Free, 64)
	if err != nil {
		fmt.Println(err)
		return false
	}

	l2, err := strconv.ParseFloat(c2.Locked, 64)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return f1+l1 > f2+l2
}

type balanceLessFunc func(p1, p2 *binance.Balance) bool

// multiSorter implements the Sort interface, sorting the changes within.
type balanceMultiSorter struct {
	balances []binance.Balance
	less     []balanceLessFunc
}

// OrderedBy returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func OrderedBy(balances []binance.Balance, less ...balanceLessFunc) sort.Interface {
	return &balanceMultiSorter{
		balances: balances,
		less:     less,
	}
}

// Len is part of sort.Interface.
func (ms *balanceMultiSorter) Len() int {
	return len(ms.balances)
}

// Swap is part of sort.Interface.
func (ms *balanceMultiSorter) Swap(i, j int) {
	ms.balances[i], ms.balances[j] = ms.balances[j], ms.balances[i]
}

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that discriminates between
// the two items (one is less than the other). Note that it can call the
// less functions twice per call. We could change the functions to return
// -1, 0, 1 and reduce the number of calls for greater efficiency: an
// exercise for the reader.
func (ms *balanceMultiSorter) Less(i, j int) bool {
	p, q := &ms.balances[i], &ms.balances[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return ms.less[k](p, q)
}
